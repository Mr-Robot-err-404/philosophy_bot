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
	printBreak()
	fmt.Println("CHANNEL_ID -> ", payload.ChannelId)
	fmt.Println("VIDEO_ID   -> ", payload.VideoId)
	fmt.Println("PUBLISHED   -> ", payload.Published.Time)

	if points < 500 {
		logs <- Log{Msg: "Insufficient quota points"}
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
		fmt.Println("Freq too low: ", c)
		return
	}
	c = 0

	scheduleJob(payload, jobs)
	cost <- UpdateQuotaPoints{value: points - 50}
}

func stateManager(initial ServerState, comms Comms) {
	state := initial
	for {
		select {
		case rd := <-comms.rd:
			rd.resp <- state

		case wr := <-comms.writeTkn:
			state.Credentials.access_token = wr.access_token

		case q := <-comms.writeWisdom:
			state.Quotes = append(state.Quotes, q.quote)

		case log := <-comms.logs:
			printLog(log)

			size := len(state.LogHistory)
			state.LogHistory = append(state.LogHistory, log)

			if size >= 100 {
				state.LogHistory = state.LogHistory[1:]
			}
		case cost := <-comms.points:
			state.QuotaPoints = cost.value
		}
	}
}

func serverCronJob(wr chan<- WriteAccessToken, logs chan<- Log) {
	ticker := time.NewTicker(50 * time.Minute)
	for {
		<-ticker.C
		access_token, err := refresh_token()

		if err != nil {
			logs <- Log{Err: err}
			return
		}
		update := WriteAccessToken{access_token: access_token, resp: make(chan bool)}
		wr <- update

		logs <- Log{Msg: fmt.Sprintf("%s", "Updated refresh token")}
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

func receiveJobs(jobs <-chan Worker, ch chan<- TaskResult, rd chan ReadReq, logs chan<- Log) {
	for task := range jobs {
		state := readServerState(rd)

		curr := task.Payload
		videoId := curr.VideoId
		channelId := curr.ChannelId

		if curr.Err != nil {
			logs <- Log{Err: fmt.Errorf("Task err: %s", curr.Err.Error())}
			continue
		}
		stack := shuffleStack(state.Quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		logs <- Log{Msg: fmt.Sprintf("Received task: %v", info)}

		go executeTask(ch, info, state.Credentials, task.Delay)
	}
}

func receiveTaskResults(ch <-chan TaskResult, logs chan<- Log) {
	for result := range ch {
		if result.Err != nil {
			logs <- Log{Err: fmt.Errorf("Task failed successfully: %s", result.Err.Error())}
			continue
		}
		params := database.CreateCommentParams{ID: result.Id, QuoteID: result.Info.QuoteId}
		saved, err := queries.CreateComment(ctx, params)

		if err != nil {
			logs <- Log{Err: err}
			continue
		}
		logs <- Log{Msg: fmt.Sprintf("Posted comment: %v", saved)}
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
