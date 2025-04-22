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

func evaluateXMLData(data string, points int, cfg *Config) {
	payload := parseXML(data)
	comms := cfg.comms
	dbComms := cfg.dbComms

	if payload.Err != nil {
		comms.logs <- Log{Err: payload.Err}
		return
	}
	published := payload.Published.Time
	elapsed := time.Since(payload.Published.Time)

	if elapsed > Threshold {
		comms.logs <- Log{Msg: fmt.Sprintf("Published long ago: %v | %v | %v", payload.VideoId, published, elapsed)}
		return
	}
	printBreak()
	fmt.Println("CHANNEL_ID -> ", payload.ChannelId)
	fmt.Println("VIDEO_ID   -> ", payload.VideoId)
	fmt.Println("PUBLISHED   -> ", payload.Published.Time)

	if points < 500 {
		comms.logs <- Log{Msg: fmt.Sprintf("Insufficient quota points: %d", points)}
		return
	}
	resp := findChannel(payload.ChannelId, dbComms.rd.get)

	if resp.err != nil {
		comms.logs <- Log{Err: fmt.Errorf("Couldn't find channel. ID: %s\n", payload.ChannelId)}
		return
	}
	channel := resp.channel
	c := channel.VideosSincePost + 1
	defer cleanup(&c, channel.ID, comms.logs, cfg.dbComms.seenVid)

	if c < channel.Frequency {
		comms.logs <- Log{Msg: fmt.Sprintf("Skipped for low frequency: %s", channel.Handle)}
		return
	}
	c = 0
	scheduleJob(payload, cfg.jobs)
}

func serverCronJob(comms *Comms, dbComms *DbComms) {
	refresh := time.NewTicker(50 * time.Minute)
	quota := time.NewTicker(25 * time.Hour)
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
			comms.logs <- Log{Msg: "Updated refresh token"}

		case <-trending.C:
			state := readServerState(comms.rd)

			if state.QuotaPoints < 3250 {
				comms.logs <- Log{Msg: fmt.Sprintf("Insufficient quota points for trending cron: %d", state.QuotaPoints)}
				return
			}
			wisdom := enlightenTrendingPage(comms, state)
			saveProgress(wisdom, dbComms, comms.logs)
		}
	}
}

func renewSubscription(logs chan<- Log, callback string, bearer string, dbComms *DbComms) {
	ticker := time.NewTicker(4 * 24 * time.Hour)
	for {
		<-ticker.C
		resp := getAllChannels(dbComms.rd.getAll)

		if resp.err != nil {
			logs <- Log{Err: resp.err}
			continue
		}
		subscribeToChannels(resp.channels, callback, bearer, logs)
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
		_, exists := state.Seen[videoId]
		if exists {
			comms.logs <- Log{Msg: fmt.Sprintf("Video already visited: %s", videoId)}
			continue
		}
		comms.writeSeen <- videoId

		err := simpleMan(videoId, dbComms.saveVid)
		if err != nil {
			comms.logs <- Log{Err: err}
			continue
		}
		resp := getUnusedQuotes(channelId, dbComms.rd.unused)

		if resp.err != nil {
			comms.logs <- Log{Err: fmt.Errorf("Task err: %s", resp.err.Error())}
			continue
		}
		stack := shuffleStack(resp.quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		comms.logs <- Log{Msg: fmt.Sprintf("Received task: %v", info)}
		comms.points <- UpdateQuotaPoints{value: state.QuotaPoints - COMMENT_COST}

		dbComms.saveUsage <- Usage{channelId: channelId, quoteId: q.ID}

		go executeTask(ch, info, state.Credentials, task.Delay, comms.logs)
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

func executeTask(ch chan<- TaskResult, info CommentInfo, credentials Credentials, delay time.Duration, logs chan<- Log) {
	logs <- Log{Msg: fmt.Sprintf("Sleep for %v", delay)}
	time.Sleep(delay)
	logs <- Log{Msg: fmt.Sprintf("Posting comment... | video_id: %s", info.VideoId)}
	postComment(info, credentials, ch)
}

func scheduleJob(payload HookPayload, jobs chan<- Worker) error {
	ts := helper.RndInt(MinWait, MaxWait)
	delay := time.Duration(ts) * time.Second
	jobs <- Worker{Payload: payload, Delay: delay}
	return nil
}
