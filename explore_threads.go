package main

import (
	"sort"
	"sync"
	"time"
)

type ThreadSearch struct {
	Results []ThreadItem
	Msg     string
	Err     error
	VideoId string
}

func exploreCommentThreads(key string, videos []string) ([]RankedItem, time.Duration) {
	best_comments := []RankedItem{}
	err_resp := []error{}
	ts := time.Now()

	var wg sync.WaitGroup
	ch := make(chan ThreadSearch)
	done := make(chan bool)

	for _, vid := range videos {
		wg.Add(1)
		go findCommentThread(vid, key, ch, &wg)
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
			best := rankComments(resp.Results, resp.VideoId)
			best_comments = append(best_comments, best...)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	logErrors(err_resp)

	sort.Slice(best_comments, func(i, j int) bool {
		return best_comments[i].Score > best_comments[j].Score
	})
	return filterCutoff(best_comments), time.Since(ts)
}

func filterCutoff(comments []RankedItem) []RankedItem {
	cutoff := 42
	if len(comments) == 0 || len(comments) <= cutoff {
		return comments
	}
	return comments[0:cutoff]
}
