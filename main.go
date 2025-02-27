package main

import (
	"bot/philosophy/internal/database"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

const CommentThread = "https://www.googleapis.com/youtube/v3/commentThreads"
const TrendingVideos = "https://youtube.googleapis.com/youtube/v3/videos?part=snippet&chart=mostPopular"
const PostComment = "https://www.googleapis.com/youtube/v3/comments?part=snippet"

var RegionCodes = [10]string{"GB", "AU", "US", "IE", "NL", "SE", "NO", "DK", "NZ", "ZA"}

func main() {
	cmd := flag.NewFlagSet("cmd", flag.ExitOnError)
	dev_mode := cmd.Bool("dev", false, "test")
	socrates := cmd.Bool("socrates", false, "socrates")

	cmd.Parse(os.Args[1:])

	if *dev_mode {
		sisyphus()
		b, err := os.ReadFile("./padawan.csv")
		if err != nil {
			log.Fatal(err)
		}
		csv := strings.Split(string(b), "\n")

		for i := 1; i < len(csv); i++ {
			line := csv[i]
			if len(line) == 0 {
				continue
			}
			curr := strings.Split(line, ",")
			reply, video_id := curr[0], curr[1]

			params := database.LinkVideoParams{VideoID: video_id, ID: reply}
			updated, err := queries.LinkVideo(ctx, params)

			if err != nil {
				log.Fatal(err)
			}
			fmt.Println("UPDATED -> ", updated)
		}
		return
	}
	if *socrates == false {
		fmt.Println("Diogenes lost his bowl")
		return
	}
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

	trending := searchTrendingRegions(key, vid_map)
	comments, _ := exploreCommentThreads(key, trending)

	stack := shuffleStack(table_cache.quotes)
	payload := prepareComments(comments, stack, quota)
	wisdom, ts := dropWisdom(payload, credentials)

	fmt.Println("dropped wisdom in ", ts)

	saveProgress(wisdom)
}
