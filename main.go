package main

import (
	"log"

	"github.com/joho/godotenv"
)

const CommentThread = "https://www.googleapis.com/youtube/v3/commentThreads"
const TrendingVideos = "https://youtube.googleapis.com/youtube/v3/videos?part=snippet&chart=mostPopular"
const PostComment = "https://www.googleapis.com/youtube/v3/comments?part=snippet"

var RegionCodes = [10]string{"GB", "AU", "US", "IE", "NL", "SE", "NO", "DK", "NZ", "ZA"}

func main() {
	err := connect_db("./app.db")
	if err != nil {
		log.Fatal(err)
	}
	err = godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
	err = simulateRanking()
	if err != nil {
		log.Fatal(err)
	}
	// trending := searchTrendingRegions(key)
	// exploreCommentThreads(key, trending)
	// "?part=snippet&order=relevance&maxResults=100&videoId=" + video_id + "&key=" + key
}
