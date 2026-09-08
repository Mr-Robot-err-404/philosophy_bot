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
	refresh := cmd.Bool("refresh", false, "refresh")
	start_server := cmd.Bool("server", false, "server")
	stats_mode := cmd.Bool("stats", false, "stats")
	philosophy_mode := cmd.Bool("socrates", false, "socrates")

	cmd.Parse(os.Args[1:])

	if *dev_mode {
		return
	}
	sisyphus()
	if *refresh {
		err := authenticate_account()
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("Done")
		return
	}
	credentials := getCredentials()

	cache, err := getTableCache(&credentials)
	if err != nil {
		log.Fatal(err)
	}
	if *start_server {
		config, err := extractGoogleConfig("client_secret.json")
		if err != nil {
			log.Fatal("Failed to extract google config")
		}
		email_payload := getEmailPayload()
		startup := Startup{credentials: credentials, quotes: cache.quotes, channels: cache.channels, seen: seenMap(cache.videos), likes: makeLikeMap(cache.replies), google_config: config, email_payload: email_payload, cache: cache}
		startServer(startup)
		return
	}
	if *stats_mode {
		// stats(cache, credentials.key)
		return
	}
	if !*philosophy_mode {
		log.Fatal("Diogenes lost his bowl")
	}
	exploreTrending(cache, credentials)
}
