package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

const CommentThread = "https://www.googleapis.com/youtube/v3/commentThreads?part=snippet"
const TrendingVideos = "https://youtube.googleapis.com/youtube/v3/videos?part=snippet&chart=mostPopular"
const PostComment = "https://www.googleapis.com/youtube/v3/comments?part=snippet"

var RegionCodes = [10]string{"GB", "AU", "US", "IE", "NL", "SE", "NO", "DK", "NZ", "ZA"}

func main() {
	cmd := flag.NewFlagSet("cmd", flag.ExitOnError)
	dev_mode := cmd.Bool("dev", false, "dev")
	start_server := cmd.Bool("server", false, "server")
	stats_mode := cmd.Bool("stats", false, "stats")
	philosophy_mode := cmd.Bool("socrates", false, "socrates")

	cmd.Parse(os.Args[1:])

	if *dev_mode {
		sisyphus()
		diff := time.Now().Unix() - int64(1*Month)
		ts := time.Unix(diff, 0)
		resp, err := queries.GetValidReplies(ctx, ts)

		if err != nil {
			log.Fatal(err)
		}
		for _, item := range resp {
			fmt.Println(item.CreatedAt.Format(time.RFC1123))
		}
		return
	}
	sisyphus()
	credentials := getCredentials()

	cache, err := getTableCache(&credentials.access_token)
	if err != nil {
		log.Fatal(err)
	}
	if *start_server {
		startup := Startup{credentials: credentials, quotes: cache.quotes, channels: cache.channels, seen: seenMap(cache.videos), likes: makeLikeMap(cache.replies)}
		startServer(startup)
		return
	}
	if *stats_mode {
		// stats(cache, credentials.key)
		return
	}
	if *philosophy_mode == false {
		log.Fatal("Diogenes lost his bowl")
	}
	exploreTrending(cache, credentials)
}
