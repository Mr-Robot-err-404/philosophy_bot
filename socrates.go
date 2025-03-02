package main

import "fmt"

func cronJob(cache TableCache, credentials Credentials) {
	quota := int(cache.quota.Quota)

	vid_map := makeVidMap(cache.videos)
	trending := searchTrendingRegions(credentials.key, vid_map)
	comments, _ := exploreCommentThreads(credentials.key, trending)

	stack := shuffleStack(cache.quotes)
	payload := prepareComments(comments, stack, quota)

	wisdom, ts := dropWisdom(payload, credentials)

	fmt.Println("dropped wisdom in ", ts)

	saveProgress(wisdom)
}
