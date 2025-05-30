package main

import (
	"bot/philosophy/internal/database"
	"fmt"
	"math/rand/v2"
	"time"
)

const COMMENT_COST = 50

type JsonLog struct {
	Msg string `json:"msg,omitempty"`
	Err string `json:"err,omitempty"`
	Ts  string `json:"time"`
}

func printLog(log Log) {
	if log.Err != nil {
		fmt.Println(log.Err)
		return
	}
	if len(log.Msg) == 0 {
		return
	}
	fmt.Println(log.Msg)
}

func seenMap(vids []string) map[string]bool {
	seen := make(map[string]bool)
	for _, id := range vids {
		seen[id] = true
	}
	return seen
}

func recentLogs(slice []Log) []JsonLog {
	reversed := slice
	start := 0
	end := len(slice) - 1

	for start < end && start < len(reversed) && end >= 0 {
		tmp := reversed[start]
		reversed[start] = reversed[end]
		reversed[end] = tmp

		start++
		end--
	}
	return JsonLogs(reversed)
}

func JsonLogs(logs []Log) []JsonLog {
	slice := make([]JsonLog, len(logs))

	for i, log := range logs {
		ts := log.Ts.Format(time.RFC1123)
		if log.Err != nil {
			slice[i] = JsonLog{Err: log.Err.Error(), Ts: ts}
			continue
		}
		slice[i] = JsonLog{Msg: log.Msg, Ts: ts}
	}
	return slice
}

func filter(arr []string, vid_map map[string]bool) []string {
	slice := []string{}
	seen := make(map[string]string)

	for _, s := range arr {
		curr, exists := seen[s]
		if exists {
			continue
		}
		_, exists = vid_map[s]
		if exists {
			continue
		}
		seen[s] = curr
		slice = append(slice, s)
	}
	return slice
}

func makeLikeMap[C GenericComment](comments []C) map[string]int {
	likeMap := make(map[string]int)

	for _, item := range comments {
		id := item.GetID()
		likes := item.GetLikes()
		likeMap[id] = int(likes)
	}
	return likeMap
}

func statsQuota(isReplies bool, base int) (int, int) {
	if isReplies {
		return base * 2, 50
	}
	return base, 100
}
func toggle(isReplies *bool) {
	if *isReplies {
		*isReplies = false
		return
	}
	*isReplies = true
}

func makeVidMap(videos []string) map[string]bool {
	vid_map := make(map[string]bool)

	for _, id := range videos {
		_, exists := vid_map[id]
		if exists {
			continue
		}
		vid_map[id] = true
	}
	return vid_map
}

func simpleCSV(vids []string) string {
	csv := ""
	for _, s := range vids {
		csv += s + "\n"
	}
	return csv
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func reduceReplies[C GenericComment](comments []C) []string {
	arr := []string{}

	for _, curr := range comments {
		arr = append(arr, curr.GetID())
	}
	return arr
}

func constructWisdom(q string, author string) string {
	quote := "Beep, bop... I'm the Philosophy Bot. Here, have a quote: \n\n" + `"`
	quote += q + `"` + "\n"
	quote += "~ " + author
	return quote
}

func shuffleStack(quotes []database.Cornucopium) []database.Cornucopium {
	stack := quotes[0:]
	rand.Shuffle(len(stack), func(i, j int) {
		tmp := stack[i]
		stack[i] = stack[j]
		stack[j] = tmp
	})
	return stack
}

func idxOf(arr string, target byte) int {
	for i := range arr {
		if arr[i] == target {
			return i
		}
	}
	return -1
}

func logErrors(slice []error) {
	for _, err := range slice {
		fmt.Println(err)
	}
}

func printBreak() {
	fmt.Println("---------")
}
