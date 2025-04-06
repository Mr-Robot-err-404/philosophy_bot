package main

import (
	"bot/philosophy/internal/database"
	"fmt"
	"math/rand/v2"
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

func recentLogs(slice []Log) []JsonLog {
	reversed := slice
	start := 0
	end := len(slice) - 1

	for start < end && start < len(slice) && end >= 0 {
		tmp := reversed[start]
		reversed[start] = slice[end]
		reversed[end] = tmp

		start++
		end--
	}
	return JsonLogs(reversed)
}

func JsonLogs(logs []Log) []JsonLog {
	slice := make([]JsonLog, len(logs))
	for i, log := range logs {
		slice[i] = JsonLog{Msg: log.Msg, Ts: log.Ts.String(), Err: log.Err.Error()}
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

func makeLikeMap(replies []database.Reply) map[string]int {
	likeMap := make(map[string]int)

	for _, item := range replies {
		id := item.ID
		likes := item.Likes
		likeMap[id] = int(likes)
	}
	return likeMap
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

func reduceReplies(replies []database.Reply) []string {
	arr := []string{}

	for _, curr := range replies {
		arr = append(arr, curr.ID)
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
