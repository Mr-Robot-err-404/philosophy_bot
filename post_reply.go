package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func postReply(payload ReplyPayload, credentials Credentials) (PostedReplyResp, error) {
	var comment_resp PostedReplyResp

	url := PostComment + "&key=" + credentials.key
	json_body, err := json.Marshal(&payload)
	if err != nil {
		return comment_resp, err
	}
	body_reader := bytes.NewReader(json_body)
	req, err := http.NewRequest(http.MethodPost, url, body_reader)

	req.Header.Set("Authorization", "Bearer "+credentials.access_token)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return comment_resp, err
	}
	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return comment_resp, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return comment_resp, fmt.Errorf("%s\n", resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return comment_resp, err
	}
	err = json.Unmarshal(body, &comment_resp)
	return comment_resp, err

}
