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
	table_cache, err := getTableCache()
	if err != nil {
		log.Fatal(err)
	}
	// access_token := os.Getenv("ACCESS_TOKEN")
	key := os.Getenv("QUOTE_API_KEY")
	vid_map := makeVidMap(table_cache.videos)

	// stack := shuffleStack(table_cache.quotes)
	trending := searchTrendingRegions(key, vid_map)
	comments := exploreCommentThreads(key, trending)

	for _, ranked := range comments {
		fmt.Println("VIDEO -> ", ranked.Item.Snippet.VideoId)
		fmt.Println("SCORE -> ", ranked.Score)
		fmt.Println("LIKES -> ", ranked.Item.Snippet.TopLevelComment.Snippet.LikeCount)
		fmt.Println("REPLY -> ", ranked.Item.Snippet.TotalReplyCount)
		fmt.Println("----------")
	}
}
