package main

import (
	"fmt"
	"time"
)

type Summary struct {
	ts    time.Duration
	title string
	end   bool
}

func cronJob(cache TableCache, credentials Credentials) {
	quota := int(cache.quota.Quota)
	vid_map := makeVidMap(cache.videos)

	trending, ts := searchTrendingRegions(credentials.key, vid_map)
	printSummary(Summary{ts: ts, title: "Explored Trending Videos"})

	comments, ts := exploreCommentThreads(credentials.key, trending)
	printSummary(Summary{ts: ts, title: "Explored Comment Threads"})

	stack := shuffleStack(cache.quotes)
	payload := prepareComments(comments, stack, quota)

	wisdom, ts := dropWisdom(payload, credentials)
	printSummary(Summary{ts: ts, title: "Dropped wisdom", end: true})

	saveProgress(wisdom)
}

func printSummary(summary Summary) {
	fmt.Println("---------")
	fmt.Printf("%s in: %v\n", summary.title, summary.ts)
	if summary.end {
		fmt.Println("---------")
	}
}
