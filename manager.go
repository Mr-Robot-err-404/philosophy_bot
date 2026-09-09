package main

import (
	"bot/philosophy/internal/database"
	"database/sql"
	"fmt"
	"time"
)

const Month = 30 * 24 * 3600

const MaxLogHistory = 1000

type rdChannelResp struct {
	channel database.Channel
	err     error
}
type FindTag struct {
	resp  chan rdChannelResp
	value string
}
type DbReadComms struct {
	findTag         chan FindTag
	get             chan GetChannel
	getAll          chan GetAll
	unused          chan GetUnused
	replies         chan GetReplies
	comments        chan GetComments
	popularComments chan PopularComments
	popularReplies  chan PopularReplies
}
type GetComments struct{ resp chan CommentResp }
type CommentResp struct {
	err      error
	comments []Comment
}
type GetReplies struct{ resp chan RepliesResp }
type RepliesResp struct {
	err     error
	replies []Reply
}
type GetUnused struct {
	id   string
	resp chan UnusedResp
}
type UnusedResp struct {
	quotes []database.Cornucopium
	err    error
}

type PopularComments = struct{ resp chan PopularCommentsResp }
type PopularCommentsResp struct {
	err      error
	comments []database.GetPopularCommentsRow
}

type PopularReplies = struct{ resp chan PopularRepliesResp }
type PopularRepliesResp struct {
	err     error
	replies []database.GetPopularRepliesRow
}
type GetAll struct{ resp chan GetAllResp }

type GetAllResp struct {
	channels []database.Channel
	err      error
}
type GetChannel struct {
	id   string
	resp chan GetResp
}

type GetResp struct {
	err     error
	channel database.Channel
}

func stateManager(initial ServerState, comms *Comms, dbComms *DbComms) {
	state := initial
	for {
		select {
		case rd := <-comms.rd:
			rd.resp <- state

		case wr := <-comms.writeTkn:
			state.Credentials.access_token = wr.token

		case refresh := <-comms.refreshTkn:
			state.Credentials.refresh_token = refresh.token

		case wisdom := <-comms.writeWisdom:
			state.Quotes = append(state.Quotes, wisdom.quote)

		case id := <-comms.writeSeen:
			state.Seen[id] = true

		case log := <-comms.logs:
			log.Ts = time.Now()
			printLog(log)

			if len(state.LogHistory) < MaxLogHistory {
				state.LogHistory = append(state.LogHistory, log)
				continue
			}
			copy(state.LogHistory, state.LogHistory[1:])
			state.LogHistory[MaxLogHistory-1] = log
		case tick := <-comms.schedule:
			if tick.every == 0 {
				state.Schedule[tick.job] = time.Time{}
				continue
			}
			state.Schedule[tick.job] = time.Now().Add(tick.every)

		case quota := <-comms.points:
			state.QuotaPoints = quota.value
			dbComms.saveQuota <- quota.value

		case spend := <-comms.spend:
			state.QuotaPoints -= spend.cost

			if state.QuotaPoints < 0 {
				state.QuotaPoints = 0
			}
			dbComms.saveQuota <- state.QuotaPoints
		}
	}
}

func dbManager(comms *DbComms, logs chan<- Log) {
	for {
		select {
		case vid := <-comms.saveVid:
			_, err := queries.SaveVideo(ctx, vid.id)
			vid.resp <- err

		case reply := <-comms.saveReply:
			saved, err := queries.StoreReply(ctx, reply.params)
			reply.resp <- ReplyResp{reply: saved, err: err}

		case params := <-comms.saveComment:
			saved, err := queries.CreateComment(ctx, params)
			if err != nil {
				logs <- Log{Scope: "db", Msg: "Failed to save comment", Err: err}
				continue
			}
			logs <- Log{Scope: "db", Msg: fmt.Sprintf("Saved comment -> %s", saved.ID)}

		case wisdom := <-comms.wisdom:
			quote, err := queries.CreateQuote(ctx, wisdom.epiphany)
			wisdom.resp <- WisdomResp{quote: quote, err: err}

		case get := <-comms.rd.get:
			channel, err := queries.FindChannel(ctx, get.id)
			get.resp <- GetResp{channel: channel, err: err}

		case getAll := <-comms.rd.getAll:
			channels, err := queries.GetChannels(ctx)
			getAll.resp <- GetAllResp{channels: channels, err: err}

		case unused := <-comms.rd.unused:
			quotes, err := queries.SelectUnusedQuotes(ctx, unused.id)
			unused.resp <- UnusedResp{quotes: quotes, err: err}

		case comments := <-comms.rd.comments:
			diff := time.Now().Unix() - int64(3*Month)
			ts := time.Unix(diff, 0)
			resp, err := queries.GetValidComments(ctx, ts)

			if err != nil {
				comments.resp <- CommentResp{err: err}
				continue
			}
			comments.resp <- CommentResp{comments: convertComments(resp)}

		case replies := <-comms.rd.replies:
			diff := time.Now().Unix() - int64(3*Month)
			ts := time.Unix(diff, 0)
			resp, err := queries.GetValidReplies(ctx, ts)

			if err != nil {
				replies.resp <- RepliesResp{err: err}
				continue
			}
			replies.resp <- RepliesResp{replies: convertReplies(resp)}

		case popular := <-comms.rd.popularComments:
			comments, err := queries.GetPopularComments(ctx)
			popular.resp <- PopularCommentsResp{comments: comments, err: err}

		case rabbitHole := <-comms.rd.popularReplies:
			replies, err := queries.GetPopularReplies(ctx)
			rabbitHole.resp <- PopularRepliesResp{replies: replies, err: err}

		case seen := <-comms.seenVid:
			_, err := queries.UpdateVideosSincePost(ctx, seen.params)
			seen.resp <- err

		case create := <-comms.createChannel:
			channel, err := queries.CreateChannel(ctx, create.params)
			if err != nil {
				create.resp <- CreateResp{err: err}
				continue
			}
			create.resp <- CreateResp{channel: channel}

		case del := <-comms.deleteChannel:
			_, err := queries.DeleteChannel(ctx, del.id)
			del.resp <- err

		case usage := <-comms.saveUsage:
			params := database.SaveUsageParams{ChannelID: usage.channelId, QuoteID: usage.quoteId}
			_, err := queries.SaveUsage(ctx, params)

			if err != nil {
				logs <- Log{Scope: "db", Msg: "Failed to save quote usage", Err: err}
				continue
			}
			logs <- Log{Scope: "db", Msg: fmt.Sprintf("Marked quote %d used by %s", usage.quoteId, usage.channelId)}

		case quota := <-comms.saveQuota:
			n, err := queries.UpdateQuota(ctx, int64(quota))
			if err != nil {
				logs <- Log{Scope: "db", Msg: "Failed to persist quota", Err: err}
				continue
			}
			logs <- Log{Scope: "db", Msg: fmt.Sprintf("Quota now %d", n.Quota)}

		case <-comms.resetQuota:
			_, err := queries.RefreshQuota(ctx)
			if err != nil {
				logs <- Log{Scope: "db", Msg: "Failed to reset quota", Err: err}
				continue
			}
			logs <- Log{Scope: "db", Msg: "Quota reset"}

		case freq := <-comms.updateFreq:
			_, err := queries.UpdateChannelFreq(ctx, freq.params)
			freq.resp <- err

		case create := <-comms.tasks.create:
			task, err := queries.CreateTask(ctx, create.params)
			create.resp <- TaskResp{task: task, err: err}

		case due := <-comms.tasks.due:
			tasks, err := queries.GetDueTasks(ctx, due.limit)
			due.resp <- DueTasksResp{tasks: tasks, err: err}

		case claim := <-comms.tasks.claim:
			task, err := queries.ClaimTask(ctx, claim.id)
			claim.resp <- TaskResp{task: task, err: err}

		case done := <-comms.tasks.complete:
			task, err := queries.CompleteTask(ctx, done.params)
			done.resp <- TaskResp{task: task, err: err}

		case failed := <-comms.tasks.fail:
			task, err := queries.FailTask(ctx, failed.params)
			failed.resp <- TaskResp{task: task, err: err}

		case again := <-comms.tasks.retry:
			task, err := queries.RetryTask(ctx, again.params)
			again.resp <- TaskResp{task: task, err: err}

		case release := <-comms.tasks.release:
			before := sql.NullTime{Time: release.before, Valid: true}
			tasks, err := queries.ReleaseStaleTasks(ctx, before)
			release.resp <- DueTasksResp{tasks: tasks, err: err}

		case running := <-comms.tasks.running:
			tasks, err := queries.GetRunningTasks(ctx)
			running.resp <- DueTasksResp{tasks: tasks, err: err}

		case purge := <-comms.tasks.purge:
			tasks, err := queries.DeleteOldTasks(ctx, sql.NullTime{Time: purge.before, Valid: true})
			purge.resp <- DueTasksResp{tasks: tasks, err: err}

		case rdTag := <-comms.rd.findTag:
			channel, err := queries.FindTag(ctx, rdTag.value)
			rdTag.resp <- rdChannelResp{channel: channel, err: err}
		}
	}
}
