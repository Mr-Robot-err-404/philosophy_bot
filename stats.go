package main

import (
	"bot/philosophy/internal/database"
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

func updateStats(replies []database.Reply, key string, likeMap map[string]int) []UpdatedStats {
	mat := splitWorkload(replies)
	stats := []UpdatedStats{}
	err_resp := []error{}

	var wg sync.WaitGroup
	ch := make(chan StatsChan)
	done := make(chan bool)

	for _, curr := range mat {
		wg.Add(1)
		go retrieveStats(curr, key, ch, &wg)
	}
	go func() {
		for {
			resp, next := <-ch
			if !next {
				done <- true
				continue
			}
			if resp.Err != nil {
				err_resp = append(err_resp, resp.Err)
				continue
			}
			updates := compareLikes(likeMap, resp.Data)
			stats = append(stats, updates...)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	logErrors(err_resp)
	return stats
}

func retrieveStats(replies []database.Reply, key string, ch chan<- StatsChan, wg *sync.WaitGroup) {
	defer wg.Done()
	var stats Stats

	payload := reduceReplies(replies)
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

func stats(cache TableCache, key string) {
	likeMap := makeLikeMap(cache.replies)
	stats := updateStats(cache.replies, key, likeMap)
	_, err_resp := updateLikes(stats)

	logErrors(err_resp)

	fmt.Println("Updated stats: ", len(stats))
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

func splitWorkload(replies []database.Reply) [][]database.Reply {
	mat := [][]database.Reply{}
	const width = 50

	if len(replies) == 0 {
		return mat
	}
	start := 0
	end := width

	for {
		chunk := replies[start:end]
		mat = append(mat, chunk)

		if end >= len(replies) {
			break
		}
		start = end
		end += width
	}
	return mat
}
