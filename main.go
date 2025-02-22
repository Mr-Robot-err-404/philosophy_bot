package main

import (
	"fmt"
	"log"
	"os"
)

const CommentThread = "https://www.googleapis.com/youtube/v3/commentThreads"
const TrendingVideos = "https://youtube.googleapis.com/youtube/v3/videos?part=snippet&chart=mostPopular"
const PostComment = "https://www.googleapis.com/youtube/v3/comments?part=snippet"

var RegionCodes = [10]string{"GB", "AU", "US", "IE", "NL", "SE", "NO", "DK", "NZ", "ZA"}

func main() {
	sisyphus()
	access_token := os.Getenv("ACCESS_TOKEN")
	key := os.Getenv("QUOTE_API_KEY")

	err := refreshAndRenewToken(&access_token)
	if err != nil {
		log.Fatal(err)
	}

	table_cache, err := getTableCache(&access_token)
	if err != nil {
		log.Fatal(err)
	}
	credentials := Credentials{key: key, access_token: access_token}
	quota := int(table_cache.quota.Quota)
	vid_map := makeVidMap(table_cache.videos)

	stack := shuffleStack(table_cache.quotes)
	trending := searchTrendingRegions(key, vid_map)
	comments, _ := exploreCommentThreads(key, trending)

	payload := prepareComments(comments, stack, quota)
	wisdom, ts := dropWisdom(payload, credentials)

	fmt.Println("dropped wisdom in ", ts, " sec")

	saveProgress(wisdom)
}
