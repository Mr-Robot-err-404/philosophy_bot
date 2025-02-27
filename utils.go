package main

import (
	"bot/philosophy/internal/database"
	"fmt"
	"math/rand/v2"
)

const COMMENT_COST = 50

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

func filterUnique(arr []string) []string {
	seen := make(map[string]bool)
	unique := []string{}

	for _, s := range arr {
		_, exists := seen[s]
		if exists {
			continue
		}
		seen[s] = true
		unique = append(unique, s)
	}
	return unique
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

func logSlice(slice []string) {
	for _, s := range slice {
		fmt.Println(s)
	}
}
