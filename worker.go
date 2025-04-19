package main

import (
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/helper"
	"fmt"
	"time"
)

type Worker struct {
	Payload HookPayload
	Delay   time.Duration
}

type TaskResult struct {
	Info CommentInfo
	Id   string
	Err  error
}

type ReadReq struct {
	resp chan ServerState
}
type WriteAccessToken struct {
	access_token string
	resp         chan bool
}
type WriteQuote struct {
	quote database.Cornucopium
	resp  chan bool
}
type UpdateQuotaPoints struct {
	value int
	resp  chan bool
}
type Log struct {
	Msg string
	Err error
	Ts  time.Time
}

func evaluateXMLData(data string, jobs chan Worker, logs chan Log, points int, cost chan UpdateQuotaPoints) {
	payload := parseXML(data)

	if payload.Err != nil {
		logs <- Log{Err: payload.Err}
		return
	}
	published := payload.Published.Time
	elapsed := time.Since(payload.Published.Time)

	if elapsed > Threshold {
		logs <- Log{Msg: fmt.Sprintf("Published long ago: %v | %v | %v", published, elapsed)}
		return
	}
	printBreak()
	fmt.Println("CHANNEL_ID -> ", payload.ChannelId)
	fmt.Println("VIDEO_ID   -> ", payload.VideoId)
	fmt.Println("PUBLISHED   -> ", payload.Published.Time)

	if points < 500 {
		logs <- Log{Msg: fmt.Sprintf("Insufficient quota points: %d", points)}
		return
	}
	channel, err := queries.FindChannel(ctx, payload.ChannelId)

	if err != nil {
		logs <- Log{Err: fmt.Errorf("Couldn't find channel. ID: %s\n", payload.ChannelId)}
		return
	}
	c := channel.VideosSincePost + 1
	defer cleanup(&c, channel.ID)

	if c < channel.Frequency {
		logs <- Log{Msg: fmt.Sprintf("Skipped for low frequency: %s", channel.Handle)}
		return
	}
	c = 0

	scheduleJob(payload, jobs)
	cost <- UpdateQuotaPoints{value: points - 50}
}

func dbManager(comms *DbComms, logs chan<- Log) {
	for {
		select {
		case id := <-comms.saveVid:
			vid, err := queries.SaveVideo(ctx, id)
			if err != nil {
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: fmt.Sprintf("Saved vid: %s", vid)}

		case params := <-comms.saveComment:
			saved, err := queries.CreateComment(ctx, params)
			if err != nil {
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: fmt.Sprintf("Posted comment: %v", saved)}

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
			logs <- Log{Msg: fmt.Sprintf("Updated Quota: %v", n)}

		case <-comms.resetQuota:
			n, err := queries.RefreshQuota(ctx)
			if err != nil {
				logs <- Log{Err: err}
				continue
			}
			logs <- Log{Msg: fmt.Sprintf("Reset Quota: %v", n)}
		}
	}
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

func serverCronJob(comms *Comms) {
	refresh := time.NewTicker(50 * time.Minute)
	quota := time.NewTicker(25 * time.Hour)
	// NOTE: change back to hour
	trending := time.NewTicker(30 * time.Minute)

	for {
		select {
		case <-quota.C:
			comms.points <- UpdateQuotaPoints{value: 10000}

		case <-refresh.C:
			access_token, err := refresh_token()

			if err != nil {
				comms.logs <- Log{Err: err}
				return
			}
			update := WriteAccessToken{access_token: access_token, resp: make(chan bool)}
			comms.writeTkn <- update
			comms.logs <- Log{Msg: fmt.Sprintf("%s", "Updated refresh token")}

		case <-trending.C:
			state := readServerState(comms.rd)

			if state.QuotaPoints < 3250 {
				comms.logs <- Log{Msg: fmt.Sprintf("Insufficient quota points: %d", state.QuotaPoints)}
				return
			}
			wisdom := enlightenTrendingPage(comms, state)

			// TODO: move to dbManager
			saveProgress(wisdom)
		}
	}
}

func renewSubscription(logs chan<- Log, callback string, bearer string) {
	ticker := time.NewTicker(4 * 24 * time.Hour)
	for {
		<-ticker.C
		channels, err := queries.GetChannels(ctx)

		if err != nil {
			logs <- Log{Err: err}
			continue
		}
		subscribeToChannels(channels, callback, bearer, logs)
	}
}

func receiveJobs(jobs <-chan Worker, ch chan<- TaskResult, comms *Comms, dbComms *DbComms) {
	for task := range jobs {
		state := readServerState(comms.rd)

		curr := task.Payload
		videoId := curr.VideoId
		channelId := curr.ChannelId

		if curr.Err != nil {
			comms.logs <- Log{Err: fmt.Errorf("Task err: %s", curr.Err.Error())}
			continue
		}
		quotes, err := queries.SelectUnusedQuotes(ctx, channelId)

		if err != nil {
			comms.logs <- Log{Err: fmt.Errorf("Task err: %s", curr.Err.Error())}
			continue
		}
		stack := shuffleStack(quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		comms.logs <- Log{Msg: fmt.Sprintf("Received task: %v", info)}
		comms.points <- UpdateQuotaPoints{value: state.QuotaPoints - COMMENT_COST}

		dbComms.saveUsage <- Usage{channelId: channelId, quoteId: q.ID}

		go executeTask(ch, info, state.Credentials, task.Delay)
	}
}

func receiveTaskResults(ch <-chan TaskResult, logs chan<- Log, dbComms *DbComms) {
	for result := range ch {
		if result.Err != nil {
			logs <- Log{Err: fmt.Errorf("Task failed successfully: %s", result.Err.Error())}
			continue
		}
		params := database.CreateCommentParams{ID: result.Id, QuoteID: result.Info.QuoteId}
		dbComms.saveComment <- params
	}
}

func executeTask(ch chan<- TaskResult, info CommentInfo, credentials Credentials, delay time.Duration) {
	time.Sleep(delay)
	postComment(info, credentials, ch)
}

func scheduleJob(payload HookPayload, jobs chan<- Worker) error {
	fmt.Println("job scheduled")
	ts := helper.RndInt(MinWait, MaxWait)
	delay := time.Duration(ts) * time.Second
	jobs <- Worker{Payload: payload, Delay: delay}
	return nil
}
