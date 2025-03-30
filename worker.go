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

type WriteCredentials struct {
	credentials Credentials
	resp        chan bool
}
type WriteQuote struct {
	quote database.Cornucopium
	resp  chan bool
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

func stateManager(initial ServerState, read chan ReadReq, wrCreds chan WriteCredentials, wrQuote chan WriteQuote) {
	state := initial
	for {
		select {
		case rd := <-read:
			rd.resp <- state

		case wr := <-wrCreds:
			state.Credentials = wr.credentials
			wr.resp <- true

		case q := <-wrQuote:
			state.Quotes = append(state.Quotes, q.quote)
			q.resp <- true
		}
	}
}

func receiveJobs(jobs <-chan Worker, ch chan<- TaskResult, rd chan ReadReq) {
	for task := range jobs {
		var req ReadReq
		rd <- req
		resp := <-req.resp

		curr := task.Payload
		videoId := curr.VideoId
		channelId := curr.ChannelId

		if curr.Err != nil {
			fmt.Println("TASK ERR -> ", curr.Err)
			continue
		}
		stack := shuffleStack(resp.Quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		fmt.Println("RECEIVED TASK -> ", info)

		go executeTask(ch, info, resp.Credentials, task.Delay)
	}
}

func receiveTaskResults(ch <-chan TaskResult) {
	for result := range ch {
		if result.Err != nil {
			fmt.Println("RESULT ERR -> ", result.Err)
			continue
		}
		params := database.CreateCommentParams{ID: result.Id, QuoteID: result.Info.QuoteId}
		saved, err := queries.CreateComment(ctx, params)

		if err != nil {
			fmt.Println("SAVING FAIL: ", err)
			continue
		}
		fmt.Println("SAVED -> ", saved)
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
