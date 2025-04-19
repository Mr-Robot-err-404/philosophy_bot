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
	payload := prepareComments(comments, stack, quota)

	wisdom, dur := dropWisdom(payload, credentials)
	printSummary(Summary{dur: dur, title: "Dropped wisdom", end: true})

	saveProgress(wisdom)
}

func enlightenTrendingPage(comms *Comms, state ServerState) []WiseReply {
	cost := SearchCost
	videos, dur := searchTrendingRegions(state.Credentials.key, state.Seen)

	summary := makeSummary(Summary{dur: dur, title: "Explored Trending Videos"})
	comms.logs <- Log{Msg: summary}

	comments, dur := exploreCommentThreads(state.Credentials.key, videos)
	cost += len(videos)

	summary = makeSummary(Summary{dur: dur, title: fmt.Sprintf("Explored %d Comment Threads", len(videos))})
	comms.logs <- Log{Msg: summary}

	stack := shuffleStack(state.Quotes)
	payload := prepareComments(comments, stack, state.QuotaPoints)
	wisdom, dur := dropWisdom(payload, state.Credentials)
	cost += COMMENT_COST * len(comments)

	summary = makeSummary(Summary{dur: dur, title: fmt.Sprintf("Dropped %d pieces of wisdom")})
	comms.logs <- Log{Msg: summary}
	comms.points <- UpdateQuotaPoints{value: state.QuotaPoints - cost}

	return wisdom
}

func makeSummary(summary Summary) string {
	return fmt.Sprintf("%s in: %v\n", summary.title, summary.dur)
}

func printSummary(summary Summary) {
	printBreak()
	fmt.Printf("%s in: %v\n", summary.title, summary.dur)
	if summary.end {
		printBreak()
	}
}
