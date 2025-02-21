package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type ReplyPayload struct {
	Snippet struct {
		ParentId     string `json:"parentId"`
		TextOriginal string `json:"textOriginal"`
	} `json:"snippet"`
}

type PostedReplyResp struct {
	Id      string `json:"id"`
	Snippet struct {
		ChannelId    string `json:"channelId"`
		TextOriginal string `json:"textOriginal"`
		ParentId     string `json:"parentId"`
	} `json:"snippet"`
}

type Credentials struct {
	key          string
	access_token string
}

func postReply(payload ReplyPayload, credentials Credentials, ch chan<- WiseReply, wg *sync.WaitGroup) {
	defer wg.Done()
	var comment_resp PostedReplyResp

	url := PostComment + "&key=" + credentials.key
	json_body, err := json.Marshal(&payload)
	if err != nil {
		ch <- WiseReply{Err: err}
		return
	}
	body_reader := bytes.NewReader(json_body)
	req, err := http.NewRequest(http.MethodPost, url, body_reader)

	req.Header.Set("Authorization", "Bearer "+credentials.access_token)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		ch <- WiseReply{Err: err}
		return

	}
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		ch <- WiseReply{Err: err}
		return
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		ch <- WiseReply{Err: fmt.Errorf("%s\n", resp.Status)}
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ch <- WiseReply{Err: err}
		return
	}
	err = json.Unmarshal(body, &comment_resp)
	if err != nil {
		ch <- WiseReply{Err: err}
		return
	}
	ch <- WiseReply{Resp: comment_resp}
}
