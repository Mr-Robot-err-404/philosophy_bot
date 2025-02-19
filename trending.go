package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type TrendingResp struct {
	Items []TrendingItem `json:"items"`
}

type TrendingItem struct {
	Id      string `json:"id"`
	Snippet struct {
		ChannelId  string `json:"channelId"`
		CategoryId string `json:"categoryId"`
		Title      string `json:"title"`
	} `json:"snippet"`
}

func reduceItems(items []TrendingItem) []string {
	arr := []string{}
	for _, item := range items {
		arr = append(arr, item.Id)
	}
	return arr
}

func saveVisitedVids(vids []string) []error {
	err_resp := []error{}
	for _, id := range vids {
		_, err := queries.SaveVideo(ctx, id)
		if err != nil {
			err_resp = append(err_resp, err)
		}
	}
	return err_resp
}

func getTrendingVideos(regionCode string, key string, ch chan<- VideoSearch, wg *sync.WaitGroup) {
	defer wg.Done()
	var trending TrendingResp

	url := TrendingVideos + "&regionCode=" + regionCode + "&key=" + key
	resp, err := http.Get(url)
	if err != nil {
		ch <- VideoSearch{Err: err}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		ch <- VideoSearch{Err: err}
		return
	}
	err = json.Unmarshal(body, &trending)
	if err != nil {
		ch <- VideoSearch{Err: err}
		return
	}
	msg := fmt.Sprintf("fetched -> %s\n", regionCode)
	ch <- VideoSearch{Results: reduceItems(trending.Items), Msg: msg}
}
