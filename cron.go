package main

import (
	"fmt"
	"sync"
	"time"
)

type ThreadSearch struct {
	Results []ThreadItem
	Msg     string
	Err     error
}

func exploreCommentThreads(key string, videos []string) {
	total_results := []ThreadItem{}
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
			total_results = append(total_results, resp.Results...)
		}
	}()
	wg.Wait()
	close(ch)
	<-done

	// TODO: build a ranking system such that the best comments are returned

	elapsed := time.Since(ts)

	fmt.Println("ELAPSED : ", elapsed)
	fmt.Println("COMMENTS: ", len(total_results))
	fmt.Println("VIDEOS  : ", len(videos))
}
