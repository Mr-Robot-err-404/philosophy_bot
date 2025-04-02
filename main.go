package main

import (
	"flag"
	"fmt"
	"log"
	"os"
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
		file, err := os.ReadFile("./tmp/xml/cd2e86ea-183a-45be-89f3-7eb33caa8ac8.xml")
		if err != nil {
			log.Fatal(err)
		}
		payload := parseXML(string(file))

		if payload.Err != nil {
			fmt.Println(payload.Err)
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
		startServer(credentials, cache.quotes, cache.channels)
		return
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
