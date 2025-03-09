package main

import (
	"bytes"
	"encoding/json"
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
type CommentResp struct {
	Id string `json:"id"`
}

func postComment(info CommentInfo, credentials Credentials) (CommentResp, error) {
	var data CommentResp

	url := CommentThread + "&key=" + credentials.key
	body, err := json.Marshal(&info.Payload)

	if err != nil {
		return data, err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return data, err
	}
	client := http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		return data, err
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(req.Body)
	if err != nil {
		return data, err
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return data, err
	}
	return data, nil
}
