package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const Channel = "https://youtube.googleapis.com/youtube/v3/channels?part=snippet&forHandle=%40"

type ChannelResp struct {
	Items []ChannelItem `json:"items"`
}

type ChannelItem struct {
	Id      string `json:"id"`
	Snippet struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		CustomUrl   string `json:"customUrl"`
	} `json:"snippet"`
}

func getChannel(s string, key string) (ChannelItem, error) {
	tag := trimTag(s)
	q := url.PathEscape(tag)

	url := Channel + q + "&key=" + key
	resp, err := http.Get(url)

	if err != nil {
		return ChannelItem{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return ChannelItem{}, fmt.Errorf("%s\n", resp.Status)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return ChannelItem{}, err
	}
	var channel ChannelResp
	err = json.Unmarshal(body, &channel)

	if err != nil {
		return ChannelItem{}, err
	}
	return channel.Items[0], nil
}

func trimTag(tag string) string {
	if len(tag) < 2 {
		return tag
	}
	if tag[0] == '@' {
		return tag[1:]
	}
	return tag
}
