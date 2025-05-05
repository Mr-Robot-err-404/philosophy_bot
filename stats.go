package main

import (
	"bot/philosophy/internal/helper"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type Stats struct {
	Items []StatsItem `json:"items"`
}

type StatsItem struct {
	Id      string       `json:"id"`
	Snippet StatsSnippet `json:"snippet"`
}

type StatsSnippet struct {
	ChannelId         string `json:"channelId"`
	TextOriginal      string `json:"textOriginal"`
	ParentId          string `json:"parentId"`
	AuthorDisplayName string `json:"authorDisplayName"`
	LikeCount         int    `json:"likeCount"`
	PublishedAt       string `json:"publishedAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type StatsChan struct {
	Data []StatsItem
	Err  error
}

type UpdatedStats struct {
	likes int
	id    string
}
type StatsCall struct {
	key   string
	logs  chan<- Log
	width int
}

func updateStats[C GenericComment](comments []C, info StatsCall, likeMap map[string]int) []UpdatedStats {
	mat := splitWorkload(comments, info.width)
	stats := []UpdatedStats{}

	var wg sync.WaitGroup
	ch := make(chan StatsChan)
	done := make(chan bool)

	for _, curr := range mat {
		wg.Add(1)
		go retrieveStats(curr, info.key, ch, &wg)
	}
	go func() {
		for {
			resp, next := <-ch
			if !next {
				done <- true
				continue
			}
			if resp.Err != nil {
				info.logs <- Log{Err: resp.Err}
				continue
			}
			updates := compareLikes(likeMap, resp.Data)
			stats = append(stats, updates...)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	return stats
}

func retrieveStats[C GenericComment](comments []C, key string, ch chan<- StatsChan, wg *sync.WaitGroup) {
	defer wg.Done()
	var stats Stats

	payload := reduceReplies(comments)
	url, err := helper.BuildStatsUrl(payload, key)

	if err != nil {
		ch <- StatsChan{Err: err}
		return
	}
	resp, err := http.Get(url)
	if err != nil {
		ch <- StatsChan{Err: err}
		return
	}
	if resp.StatusCode != 200 {
		ch <- StatsChan{Err: fmt.Errorf("%s\n", resp.Status)}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		ch <- StatsChan{Err: err}
		return
	}
	err = json.Unmarshal(body, &stats)
	if err != nil {
		ch <- StatsChan{Err: err}
		return
	}
	ch <- StatsChan{Data: stats.Items}
}

func getStats[C GenericComment](comments []C, key string, logs chan<- Log, width int) []UpdatedStats {
	likeMap := makeLikeMap(comments)
	info := StatsCall{key: key, logs: logs, width: width}

	return updateStats(comments, info, likeMap)

}

func stats(dbComms *DbComms, alternate *bool, info StatsCall) {
	defer toggle(alternate)

	if *alternate {
		resp := getReplies(dbComms.rd.replies)

		if resp.err != nil {
			info.logs <- Log{Err: resp.err}
			return
		}
		stats := getStats(resp.replies, info.key, info.logs, info.width)

		if len(stats) > 0 {
			storeLikes(stats, info.logs, "replies")
		}
		return
	}
	resp := getComments(dbComms.rd.comments)

	if resp.err != nil {
		info.logs <- Log{Err: resp.err}
		return
	}
	stats := getStats(resp.comments, info.key, info.logs, info.width)

	if len(stats) > 0 {
		storeLikes(stats, info.logs, "comments")
	}
}

func compareLikes(likeMap map[string]int, updates []StatsItem) []UpdatedStats {
	stats := []UpdatedStats{}

	for _, item := range updates {
		id := item.Id
		likes, exists := likeMap[id]

		if !exists {
			continue
		}
		n := item.Snippet.LikeCount
		if n > likes {
			stats = append(stats, UpdatedStats{likes: n, id: id})
		}
	}
	return stats
}

func splitWorkload[C GenericComment](comments []C, width int) [][]C {
	mat := [][]C{}

	if len(comments) == 0 {
		return mat
	}
	start := 0
	end := min(width, len(comments))

	for {
		chunk := comments[start:end]
		mat = append(mat, chunk)

		if end >= len(comments) {
			break
		}
		start = end
		end += width
	}
	return mat
}
