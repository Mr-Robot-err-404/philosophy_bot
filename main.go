package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
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
		file, err := os.ReadFile("./tmp/xml/bb536210-f268-4651-9389-9b0e57f7d444.xml")
		if err != nil {
			log.Fatal(err)
			return
		}
		endpoint := "https://" + "a8f4-81-140-55-53.ngrok-free.app" + "/diogenes/bowl"
		body_reader := bytes.NewReader(file)

		request, err := http.NewRequest(http.MethodPost, endpoint, body_reader)
		request.Header.Set("Content-Type", "application/xml")

		if err != nil {
			log.Fatal(err)
			return
		}
		client := &http.Client{}
		resp, err := client.Do(request)

		if err != nil {
			log.Fatal(err)
			return
		}
		fmt.Println(resp.Status)
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
