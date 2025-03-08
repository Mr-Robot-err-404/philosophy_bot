package main

import (
	"fmt"
	"time"
)

func cronJob(cache TableCache, credentials Credentials) {
	quota := int(cache.quota.Quota)
	vid_map := makeVidMap(cache.videos)

	trending, ts := searchTrendingRegions(credentials.key, vid_map)
	printSummary(ts, "Explored Trending Videos")

	comments, ts := exploreCommentThreads(credentials.key, trending)
	printSummary(ts, "Explored Comment Threads")

	stack := shuffleStack(cache.quotes)
	payload := prepareComments(comments, stack, quota)

	wisdom, ts := dropWisdom(payload, credentials)
	printSummary(ts, "Dropped wisdom")

	saveProgress(wisdom)
}

func printSummary(ts time.Duration, title string) {
	fmt.Println("---------")
	fmt.Printf("%s in: %v\n", title, ts)
}
