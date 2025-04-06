package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type CommentPayload struct {
	Snippet struct {
		ChannelId       string     `json:"channelId"`
		VideoId         string     `json:"videoId"`
		TopLevelComment TopComment `json:"topLevelComment"`
	} `json:"snippet"`
}

type TopComment struct {
	Snippet struct {
		TextOriginal string `json:"textOriginal"`
	} `json:"snippet"`
}

type CommentInfo struct {
	VideoId   string
	ChannelId string
	QuoteId   int64
	Payload   CommentPayload
}
type ChannelId struct {
	Id string `json:"id"`
}

func postComment(info CommentInfo, credentials Credentials, ch chan<- TaskResult) {
	var data ChannelId

	url := CommentThread + "&key=" + credentials.key
	body, err := json.Marshal(&info.Payload)

	if err != nil {
		ch <- TaskResult{Err: err}
		return
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		ch <- TaskResult{Err: err}
		return
	}
	req.Header.Set("Authorization", "Bearer "+credentials.access_token)
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		ch <- TaskResult{Err: err}
		return
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		ch <- TaskResult{Err: fmt.Errorf("%s", resp.Status)}
		return
	}
	defer resp.Body.Close()
	body, err = io.ReadAll(resp.Body)

	if err != nil {
		ch <- TaskResult{Err: err}
		return
	}
	err = json.Unmarshal(body, &data)

	if err != nil {
		ch <- TaskResult{Err: err}
		return
	}
	ch <- TaskResult{Info: info, Id: data.Id}
}
