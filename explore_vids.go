package main

import (
	"sync"
	"time"
)

type VideoSearch struct {
	Results []string
	Msg     string
	Err     error
}

const SearchCost = 1000

func searchTrendingRegions(key string, vid_map map[string]bool) ([]string, time.Duration) {
	total_results := []string{}
	err_resp := []error{}

	var wg sync.WaitGroup
	ch := make(chan VideoSearch)
	done := make(chan bool)

	ts := time.Now()

	for _, region := range RegionCodes {
		wg.Add(1)
		go getTrendingVideos(region, key, ch, &wg)
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

	logErrors(err_resp)

	final_result := filter(total_results, vid_map)
	elapsed := time.Since(ts)

	return final_result, elapsed
}
