package main

import (
	"fmt"
	"sync"
	"time"
)

type VideoSearch struct {
	Results []string
	Msg     string
	Err     error
}

func searchTrendingRegions(key string) []string {
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

	final_result := filter(total_results)
	overlap := len(total_results) - len(final_result)
	elapsed := time.Since(ts)

	fmt.Printf("Time:    %v\n", elapsed)
	fmt.Printf("Overlap: %v\n", overlap)

	return final_result
}
