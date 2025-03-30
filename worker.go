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
type Log struct {
	msg string
	err error
	ts  time.Time
}

func evaluateXMLData(data string, jobs chan Worker) {
	payload := parseXML(data)

	if !isXMLValid(payload) {
		fmt.Printf("Invalid XML payload: %v\n", payload)
		return
	}
	if payload.Err != nil {
		fmt.Println(payload.Err)
		return
	}
	channel, err := queries.FindChannel(ctx, payload.ChannelId)

	if err != nil {
		fmt.Printf("Couldn't find channel. ID: %s\n", payload.ChannelId)
		return
	}
	c := channel.VideosSincePost + 1
	defer cleanup(&c, channel.ID)

	if c < channel.Frequency {
		return
	}
	c = 0
	scheduleJob(payload, jobs)
}

func stateManager(initial ServerState, read chan ReadReq, wrCreds chan WriteAccessToken, wrQuote chan WriteQuote, logs chan Log) {
	state := initial
	for {
		select {
		case rd := <-read:
			rd.resp <- state

		case wr := <-wrCreds:
			state.Credentials.access_token = wr.access_token
			wr.resp <- true

		case q := <-wrQuote:
			state.Quotes = append(state.Quotes, q.quote)
			q.resp <- true

		case log := <-logs:
			printLog(log)

			size := len(state.LogHistory)
			state.LogHistory = append(state.LogHistory, log)

			if size >= 100 {
				state.LogHistory = state.LogHistory[1:]
			}
		}
	}
}

func serverCronJob(wr chan<- WriteAccessToken, logs chan<- Log) {
	ticker := time.NewTicker(50 * time.Minute)
	for {
		select {
		case ts := <-ticker.C:
			access_token, err := refresh_token()

			if err != nil {
				logs <- Log{err: err}
				return
			}
			update := WriteAccessToken{access_token: access_token}
			wr <- update
			<-update.resp

			logs <- Log{msg: fmt.Sprintf("Updated refresh token: %v\n", ts)}
		}
	}
}

func receiveJobs(jobs <-chan Worker, ch chan<- TaskResult, rd chan ReadReq, logs chan<- Log) {
	for task := range jobs {
		var req ReadReq
		rd <- req
		resp := <-req.resp

		curr := task.Payload
		videoId := curr.VideoId
		channelId := curr.ChannelId

		if curr.Err != nil {
			logs <- Log{err: fmt.Errorf("Task err: %s\n", curr.Err.Error()), ts: time.Now()}
			continue
		}
		stack := shuffleStack(resp.Quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		logs <- Log{msg: fmt.Sprintf("Received task: %v\n", info), ts: time.Now()}

		go executeTask(ch, info, resp.Credentials, task.Delay)
	}
}

func receiveTaskResults(ch <-chan TaskResult, logs chan<- Log) {
	for result := range ch {
		now := time.Now()

		if result.Err != nil {
			logs <- Log{err: fmt.Errorf("Task failed successfully: %s\n", result.Err.Error()), ts: now}
			continue
		}
		params := database.CreateCommentParams{ID: result.Id, QuoteID: result.Info.QuoteId}
		saved, err := queries.CreateComment(ctx, params)

		if err != nil {
			logs <- Log{err: err, ts: now}
			continue
		}
		logs <- Log{msg: fmt.Sprintf("Posted comment: %v\n", saved), ts: now}
	}
}

func executeTask(ch chan<- TaskResult, info CommentInfo, credentials Credentials, delay time.Duration) {
	time.Sleep(delay)
	postComment(info, credentials, ch)
}

func scheduleJob(payload HookPayload, jobs chan<- Worker) error {
	ts := helper.RndInt(MinWait, MaxWait)
	delay := time.Duration(ts) * time.Second
	jobs <- Worker{Payload: payload, Delay: delay}
	return nil
}
