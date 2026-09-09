package main

import (
	"bot/philosophy/email"
	"bot/philosophy/internal/database"
	"fmt"
	"time"
)

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

	state := readServerState(comms.rd)
	if _, seen := state.Seen[payload.VideoId]; seen {
		comms.logs <- Log{Scope: "hook", Msg: fmt.Sprintf("Already visited -> %s", payload.VideoId)}
		return
	}
	quotes := getUnusedQuotes(payload.ChannelId, dbComms.rd.unused)

	if quotes.err != nil {
		comms.logs <- Log{Scope: "hook", Msg: "Failed to load quotes", Err: quotes.err}
		return
	}
	if len(quotes.quotes) == 0 {
		comms.logs <- Log{Scope: "hook", Level: LevelWarn, Msg: fmt.Sprintf("Quote pool exhausted for %s", channel.Handle)}
		return
	}
	stack := shuffleStack(quotes.quotes)
	enqueueTask(payload, stack[0], &comms, &dbComms)
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

			tracked := getAllChannels(dbComms.rd.getAll)

			if tracked.err != nil {
				comms.logs <- Log{Scope: "cron", Msg: "Failed to count channels for webhook margin", Err: tracked.err}
				continue
			}
			margin := webhookMargin(len(tracked.channels))
			required := TrendingReserve + margin

			if state.QuotaPoints < required {
				comms.logs <- Log{Scope: "cron", Level: LevelWarn, Msg: fmt.Sprintf("Insufficient quota for trending -> %d/%d (reserve %d + margin %d for %d channels)", state.QuotaPoints, required, TrendingReserve, margin, len(tracked.channels))}
				continue
			}
			comms.logs <- Log{Scope: "cron", Msg: fmt.Sprintf("Trending cleared -> %d available, holding %d for %d channels", state.QuotaPoints, margin, len(tracked.channels))}
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
