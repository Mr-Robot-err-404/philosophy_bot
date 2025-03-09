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

func receiveJobs(jobs <-chan Worker, ch chan<- TaskResult, credentials *Credentials, quotes *[]database.Cornucopium) {
	for task := range jobs {
		curr := task.Payload
		videoId := curr.VideoId
		channelId := curr.ChannelId

		if curr.Err != nil {
			continue
		}
		stack := shuffleStack(*quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		go executeTask(ch, info, *credentials, task.Delay)
	}
}

func receiveTaskResults(ch <-chan TaskResult) {
	for result := range ch {
		if result.Err != nil {
			fmt.Println(result.Err)
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

func scheduleJob(payload HookPayload, jobs chan<- Worker) {
	ts := helper.RndInt(MinWait, MaxWait)
	delay := time.Duration(ts) * time.Second
	jobs <- Worker{Payload: payload, Delay: delay}
}
