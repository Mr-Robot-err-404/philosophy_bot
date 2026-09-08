package main

import (
	"bot/philosophy/email"
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
type WriteToken struct {
	token string
	resp  chan bool
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
	Msg   string
	Err   error
	Ts    time.Time
	Level LogLevel
	Scope string
}

func evaluateXMLData(data string, points int, cfg *Config) {
	payload := parseXML(data)
	comms := cfg.comms
	dbComms := cfg.dbComms

	if payload.Err != nil {
		comms.logs <- Log{Scope: "hook", Msg: "Failed to parse webhook XML", Err: payload.Err}
		return
	}
	elapsed := time.Since(payload.Published.Time)

	if elapsed > Threshold {
		comms.logs <- Log{Scope: "hook", Level: LevelWarn, Msg: fmt.Sprintf("Video too old -> %s (%v ago)", payload.VideoId, elapsed.Round(time.Minute))}
		return
	}
	if points < 500 {
		comms.logs <- Log{Scope: "hook", Level: LevelWarn, Msg: fmt.Sprintf("Insufficient quota -> %d/500", points)}
		return
	}
	resp := findChannel(payload.ChannelId, dbComms.rd.get)

	if resp.err != nil {
		comms.logs <- Log{Scope: "hook", Err: fmt.Errorf("unknown channel %s", payload.ChannelId)}
		return
	}
	channel := resp.channel
	c := channel.VideosSincePost + 1
	defer cleanup(&c, channel.ID, comms.logs, cfg.dbComms.seenVid)

	if c < channel.Frequency {
		comms.logs <- Log{Scope: "hook", Msg: fmt.Sprintf("Skipped %s -> %d/%d videos since post", channel.Handle, c, channel.Frequency)}
		return
	}
	c = 0
	scheduleJob(payload, cfg.jobs)
}

func serverCronJob(comms *Comms, dbComms *DbComms, email_payload email.Payload, schedule map[string]time.Time, done <-chan struct{}) {
	trending := time.NewTimer(time.Until(schedule["trending"]))
	refresh := time.NewTimer(time.Until(schedule["refresh"]))
	quota := time.NewTimer(time.Until(schedule["quota"]))
	statsCron := time.NewTimer(time.Until(schedule["stats"]))

	defer trending.Stop()
	defer refresh.Stop()
	defer quota.Stop()
	defer statsCron.Stop()

	alternate := false

	for {
		select {
		case <-done:
			return

		case <-quota.C:
			quota.Reset(QuotaInterval)
			comms.schedule <- ScheduleTick{job: "quota", every: QuotaInterval}
			comms.points <- UpdateQuotaPoints{value: 10000}

		case <-refresh.C:
			refresh.Reset(RefreshInterval)
			comms.schedule <- ScheduleTick{job: "refresh", every: RefreshInterval}
			state := readServerState(comms.rd)
			access_token, err := refresh_token(state.Credentials.refresh_token)

			if err != nil {
				comms.logs <- Log{Scope: "auth", Msg: "Token refresh failed, alerting operator", Err: err}

				refresh.Stop()
				comms.schedule <- ScheduleTick{job: "refresh"}

				if err := email.Send(email_payload); err != nil {
					comms.logs <- Log{Scope: "auth", Msg: "Failed to send alert email", Err: err}
					continue
				}
				comms.logs <- Log{Scope: "auth", Level: LevelWarn, Msg: "Alert email sent, refresh ticker stopped"}
				continue
			}
			if err := renewAccessToken(access_token); err != nil {
				comms.logs <- Log{Scope: "auth", Msg: "Failed to persist access token", Err: err}
			}
			update := WriteToken{token: access_token, resp: make(chan bool)}
			comms.writeTkn <- update
			comms.logs <- Log{Scope: "auth", Msg: "Renewed access token"}

		case <-trending.C:
			trending.Reset(TrendingInterval)
			comms.schedule <- ScheduleTick{job: "trending", every: TrendingInterval}
			state := readServerState(comms.rd)

			if state.QuotaPoints < 3250 {
				comms.logs <- Log{Scope: "cron", Level: LevelWarn, Msg: fmt.Sprintf("Insufficient quota for trending -> %d/%d", state.QuotaPoints, 3250)}
				continue
			}
			wisdom := enlightenTrendingPage(comms, state)
			saveProgress(wisdom, dbComms, comms.logs, comms.writeSeen)

		case <-statsCron.C:
			statsCron.Reset(StatsInterval)
			comms.schedule <- ScheduleTick{job: "stats", every: StatsInterval}
			state := readServerState(comms.rd)
			minimum, width := statsQuota(alternate, 1250)

			if state.QuotaPoints < minimum {
				comms.logs <- Log{Scope: "cron", Level: LevelWarn, Msg: fmt.Sprintf("Insufficient quota for stats -> %d/%d", state.QuotaPoints, minimum)}
				toggle(&alternate)
				continue
			}
			info := StatsCall{key: state.Credentials.key, logs: comms.logs, width: width}
			stats(dbComms, &alternate, info)
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
			comms.logs <- Log{Scope: "task", Msg: "Malformed payload", Err: curr.Err}
			continue
		}
		_, exists := state.Seen[videoId]
		if exists {
			comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Already visited -> %s", videoId)}
			continue
		}
		comms.writeSeen <- videoId

		err := simpleMan(videoId, dbComms.saveVid)
		if err != nil {
			comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to record video %s", videoId), Err: err}
			continue
		}
		resp := getUnusedQuotes(channelId, dbComms.rd.unused)

		if resp.err != nil {
			comms.logs <- Log{Scope: "task", Msg: "Failed to load quotes", Err: resp.err}
			continue
		}
		if len(resp.quotes) == 0 {
			comms.logs <- Log{Scope: "task", Level: LevelWarn, Msg: fmt.Sprintf("Quote pool exhausted for channel %s", channelId)}
			continue
		}
		stack := shuffleStack(resp.quotes)
		q := stack[0]

		payload := CommentPayload{}
		payload.Snippet.ChannelId = channelId
		payload.Snippet.VideoId = videoId
		payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(q.Quote, q.Author)

		info := CommentInfo{VideoId: videoId, ChannelId: channelId, QuoteId: q.ID, Payload: payload}
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Queued video %s -> quote %d", info.VideoId, info.QuoteId)}
		comms.points <- UpdateQuotaPoints{value: state.QuotaPoints - COMMENT_COST}

		dbComms.saveUsage <- Usage{channelId: channelId, quoteId: q.ID}

		go executeTask(ch, info, state.Credentials, task.Delay, comms.logs)
	}
}

func receiveTaskResults(ch <-chan TaskResult, logs chan<- Log, dbComms *DbComms) {
	for result := range ch {
		if result.Err != nil {
			logs <- Log{Scope: "task", Msg: "Post failed", Err: result.Err}
			continue
		}
		params := database.CreateCommentParams{ID: result.Id, QuoteID: result.Info.QuoteId}
		dbComms.saveComment <- params
	}
}

func executeTask(ch chan<- TaskResult, info CommentInfo, credentials Credentials, delay time.Duration, logs chan<- Log) {
	logs <- Log{Scope: "task", Msg: fmt.Sprintf("Waiting %v before posting", delay)}
	time.Sleep(delay)
	logs <- Log{Scope: "task", Msg: fmt.Sprintf("Posting to video %s", info.VideoId)}
	postComment(info, credentials, ch)
}

func scheduleJob(payload HookPayload, jobs chan<- Worker) error {
	ts := helper.RndInt(MinWait, MaxWait)
	delay := time.Duration(ts) * time.Second
	jobs <- Worker{Payload: payload, Delay: delay}
	return nil
}
