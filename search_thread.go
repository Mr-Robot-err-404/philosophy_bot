package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type ThreadSnippet struct {
	ChannelId       string  `json:"channelId"`
	VideoId         string  `json:"videoId"`
	TopLevelComment Comment `json:"topLevelComment"`
	TotalReplyCount int     `json:"totalReplyCount"`
	CanReply        bool    `json:"canReply"`
}

type Comment struct {
	Snippet struct {
		TextDisplay string `json:"textDisplay"`
		LikeCount   int    `json:"likeCount"`
	} `json:"snippet"`
}

type ThreadItem struct {
	Id      string        `json:"id"`
	Snippet ThreadSnippet `json:"snippet"`
}
type ThreadResp struct {
	Items []ThreadItem `json:"items"`
}

func findCommentThread(video_id string, key string, ch chan<- ThreadSearch, wg *sync.WaitGroup) {
	defer wg.Done()
	var thread ThreadResp

	url := CommentThread + "&order=relevance&maxResults=100&videoId=" + video_id + "&key=" + key
	resp, err := http.Get(url)

	if err != nil {
		ch <- ThreadSearch{Err: err}
		return
	}
	if resp.StatusCode != 200 {
		ch <- ThreadSearch{Err: fmt.Errorf("%s\n", resp.Status)}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		ch <- ThreadSearch{Err: err}
		return
	}
	err = json.Unmarshal(body, &thread)
	if err != nil {
		ch <- ThreadSearch{Err: err}
		return
	}
	ch <- ThreadSearch{Results: thread.Items, VideoId: video_id}
}
