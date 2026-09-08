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

func usage() {
	fmt.Fprint(os.Stderr, `philosophy bot

usage:
  bot <command> [flags]

commands:
  server     run the webhook server and cron jobs
  socrates   post to trending videos once, then exit
  refresh    authorise with google and save tokens
  stats      sync like counts for existing comments
  help       show this message

run "bot <command> -h" for command flags
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	args := os.Args[2:]

	switch os.Args[1] {
	case "server":
		runServer(args)
	case "socrates":
		runSocrates(args)
	case "refresh":
		runRefresh(args)
	case "stats":
		runStats(args)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func serverAddr() string {
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		return addr
	}
	return "127.0.0.1:49399"
}

func loadCache() (Credentials, TableCache) {
	credentials := getCredentials()

	cache, err := getTableCache(&credentials)
	if err != nil {
		log.Fatal(err)
	}
	return credentials, cache
}

func runRefresh(args []string) {
	fs := flag.NewFlagSet("refresh", flag.ExitOnError)
	fs.Parse(args)

	sisyphus()

	if err := authenticate_account(); err != nil {
		log.Fatal(err)
	}
}

func runServer(args []string) {
	fs := flag.NewFlagSet("server", flag.ExitOnError)
	addr := fs.String("addr", serverAddr(), "local listen address")
	fs.Parse(args)

	sisyphus()
	credentials, cache := loadCache()

	config, err := extractGoogleConfig("client_secret.json")
	if err != nil {
		log.Fatal("Failed to extract google config")
	}
	startup := Startup{
		credentials:   credentials,
		quotes:        cache.quotes,
		channels:      cache.channels,
		seen:          seenMap(cache.videos),
		likes:         makeLikeMap(cache.replies),
		google_config: config,
		email_payload: getEmailPayload(),
		cache:         cache,
		addr:          *addr,
	}
	startServer(startup)
}

func runSocrates(args []string) {
	fs := flag.NewFlagSet("socrates", flag.ExitOnError)
	fs.Parse(args)

	sisyphus()
	credentials, cache := loadCache()
	exploreTrending(cache, credentials)
}

func runStats(args []string) {
	fs := flag.NewFlagSet("stats", flag.ExitOnError)
	fs.Parse(args)

	sisyphus()
	loadCache()
	log.Fatal("stats is not wired up for cli use yet, it runs on the server cron")
}
