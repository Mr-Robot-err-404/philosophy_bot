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

func main() {
	cmd := flag.NewFlagSet("cmd", flag.ExitOnError)
	dev_mode := cmd.Bool("dev", false, "dev")
	start_server := cmd.Bool("server", false, "server")
	stats_mode := cmd.Bool("stats", false, "stats")
	philosophy_mode := cmd.Bool("socrates", false, "socrates")

	cmd.Parse(os.Args[1:])

	if *dev_mode {
		file, err := os.ReadFile("./public/payload.xml")
		if err != nil {
			log.Fatal(err)
		}
		hook_payload := parseXML(string(file))
		if hook_payload.Err != nil {
			log.Fatal(hook_payload.Err)
		}
		return
	}
	sisyphus()
	credentials := getCredentials()

	if *start_server {
		startServer(credentials)
		return
	}
	cache, err := getTableCache(&credentials.access_token)
	if err != nil {
		log.Fatal(err)
	}
	if *stats_mode {
		stats(cache, credentials.key)
		return
	}
	if *philosophy_mode == false {
		log.Fatal("Diogenes lost his bowl")
	}
	cronJob(cache, credentials)
}
