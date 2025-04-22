package main

import (
	"bot/philosophy/internal/database"
	"fmt"
)

type rdChannelResp struct {
	channel database.Channel
	err     error
}
type FindTag struct {
	resp  chan rdChannelResp
	value string
}
type DbReadComms struct {
	findTag chan FindTag
	get     chan GetChannel
	getAll  chan GetAll
	unused  chan GetUnused
}
type GetUnused struct {
	id   string
	resp chan UnusedResp
}
type UnusedResp struct {
	quotes []database.Cornucopium
	err    error
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
			state.Credentials.access_token = wr.access_token

		case wisdom := <-comms.writeWisdom:
			state.Quotes = append(state.Quotes, wisdom.quote)

		case id := <-comms.writeSeen:
			state.Seen[id] = true

		case log := <-comms.logs:
			printLog(log)
			state.LogHistory = append(state.LogHistory, log)

			if len(state.LogHistory) >= 1000 {
				state.LogHistory = state.LogHistory[1:]
			}
		case quota := <-comms.points:
			state.QuotaPoints = quota.value
			dbComms.saveQuota <- quota.value
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
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: fmt.Sprintf("Posted comment: %s", saved.ID)}

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
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: fmt.Sprintf("Saved usage. channel: %v | quote: %v", usage.channelId, usage.quoteId)}

		case quota := <-comms.saveQuota:
			n, err := queries.UpdateQuota(ctx, int64(quota))
			if err != nil {
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: fmt.Sprintf("Updated Quota -> %d", n.Quota)}

		case <-comms.resetQuota:
			_, err := queries.RefreshQuota(ctx)
			if err != nil {
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: "Reset quota"}

		case freq := <-comms.updateFreq:
			_, err := queries.UpdateChannelFreq(ctx, freq.params)
			freq.resp <- err

		case rdTag := <-comms.rd.findTag:
			channel, err := queries.FindTag(ctx, rdTag.value)
			rdTag.resp <- rdChannelResp{channel: channel, err: err}
		}
	}
}
