package main

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type ThreadSearch struct {
	Results []ThreadItem
	Msg     string
	Err     error
}

func exploreCommentThreads(key string, videos []string) []RankedItem {
	best_comments := []RankedItem{}
	err_resp := []error{}

	var wg sync.WaitGroup
	ch := make(chan ThreadSearch)
	done := make(chan bool)

	ts := time.Now()

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
			best := rankComments(resp.Results)
			best_comments = append(best_comments, best...)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	sort.Slice(best_comments, func(i, j int) bool {
		return best_comments[i].Score > best_comments[j].Score
	})
	elapsed := time.Since(ts)

	fmt.Println("ELAPSED : ", elapsed)
	fmt.Println("COMMENTS: ", len(best_comments))
	fmt.Println("VIDEOS  : ", len(videos))
	fmt.Println("--------------")

	return best_comments
}
