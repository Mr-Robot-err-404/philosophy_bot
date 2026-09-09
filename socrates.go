package main

import (
	"fmt"
	"time"
)

type Summary struct {
	dur   time.Duration
	title string
	end   bool
}

func exploreTrending(cache TableCache, credentials Credentials) {
	quota := int(cache.quota.Quota)
	vid_map := makeVidMap(cache.videos)

	trending, dur := searchTrendingRegions(credentials.key, vid_map)
	printSummary(Summary{dur: dur, title: "Explored Trending Videos"})

	comments, dur := exploreCommentThreads(credentials.key, trending)
	printSummary(Summary{dur: dur, title: "Explored Comment Threads"})

	stack := shuffleStack(cache.quotes)
	scanned := (len(RegionCodes) * LIST_COST) + (len(trending) * LIST_COST)
	payload := prepareComments(comments, stack, quota-QuotaFloor-scanned)

	wisdom, dur := dropWisdom(payload, credentials)
	printSummary(Summary{dur: dur, title: "Dropped wisdom", end: true})

	storeProgress(wisdom)
}

func enlightenTrendingPage(comms *Comms, state ServerState, budget int) []WiseReply {
	cost := len(RegionCodes) * LIST_COST
	videos, dur := searchTrendingRegions(state.Credentials.key, state.Seen)

	summary := makeSummary(Summary{dur: dur, title: fmt.Sprintf("Explored Trending Videos -> %d fresh", len(videos))})
	comms.logs <- Log{Scope: "cron", Msg: summary}

	comments, dur := exploreCommentThreads(state.Credentials.key, videos)
	cost += len(videos) * LIST_COST

	summary = makeSummary(Summary{dur: dur, title: fmt.Sprintf("Explored %d Comment Threads -> %d candidates", len(videos), len(comments))})
	comms.logs <- Log{Scope: "cron", Msg: summary}

	stack := shuffleStack(state.Quotes)
	payload := prepareComments(comments, stack, budget-cost)
	cost += COMMENT_COST * len(payload)

	wisdom, dur := dropWisdom(payload, state.Credentials)

	summary = makeSummary(Summary{dur: dur, title: fmt.Sprintf("Dropped %d of %d attempted", len(wisdom), len(payload))})
	comms.logs <- Log{Scope: "cron", Msg: summary}

	comms.spend <- Spend{cost: cost, reason: "trending"}
	comms.logs <- Log{Scope: "cron", Msg: fmt.Sprintf("Trending spent %d -> %d scan + %d posts | %d remaining", cost, len(RegionCodes)+len(videos), len(payload), state.QuotaPoints-cost)}

	return wisdom
}

func makeSummary(summary Summary) string {
	return fmt.Sprintf("%s in: %v", summary.title, summary.dur)
}

func printSummary(summary Summary) {
	printBreak()
	fmt.Printf("%s in: %v\n", summary.title, summary.dur)
	if summary.end {
		printBreak()
	}
}
