package main

import (
	"flag"
	"log"
	"os"
)

const CommentThread = "https://www.googleapis.com/youtube/v3/commentThreads"
const TrendingVideos = "https://youtube.googleapis.com/youtube/v3/videos?part=snippet&chart=mostPopular"
const PostComment = "https://www.googleapis.com/youtube/v3/comments?part=snippet"

var RegionCodes = [10]string{"GB", "AU", "US", "IE", "NL", "SE", "NO", "DK", "NZ", "ZA"}

// HACK:
//   -> bring in the webhooks and a server!
//  |
//  -> keep track of quota with a comfortable margin
//  |

func main() {
	sisyphus()

	cmd := flag.NewFlagSet("cmd", flag.ExitOnError)
	dev_mode := cmd.Bool("dev", false, "dev")
	start_server := cmd.Bool("server", false, "server")
	stats_mode := cmd.Bool("stats", false, "stats")
	philosophy_mode := cmd.Bool("socrates", false, "socrates")

	cmd.Parse(os.Args[1:])

	if *dev_mode {
		_, err := os.ReadFile("./trending_vid.csv")
		if err != nil {
			log.Fatal(err)
		}
		return
	}
	access_token := os.Getenv("ACCESS_TOKEN")
	key := os.Getenv("QUOTE_API_KEY")
	credentials := Credentials{key: key, access_token: access_token}

	if *start_server {
		startServer(credentials)
		return
	}

	cache, err := getTableCache(&access_token)
	if err != nil {
		log.Fatal(err)
	}
	if *stats_mode {
		stats(cache, key)
		return
	}
	if *philosophy_mode == false {
		log.Fatal("Diogenes lost his bowl")
	}
	cronJob(cache, credentials)
}
