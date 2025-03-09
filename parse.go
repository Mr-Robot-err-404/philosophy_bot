package main

import (
	"fmt"
	"strings"
	"time"
)

type HookPayload struct {
	ChannelId string
	VideoId   string
	Published time.Time
	Err       error
}

func parseXML(data string) HookPayload {
	var payload HookPayload
	lines := strings.Split(data, "\n")
	seen := false

	for _, curr := range lines {
		s := strings.TrimSpace(curr)

		if strings.Contains(s, "yt:videoId") {
			payload.VideoId = parseId(s)
			continue
		}
		if strings.Contains(s, "yt:channelId") {
			payload.ChannelId = parseId(s)
		}
		if strings.Contains(s, "published") {
			seen = true
			str := parseId(s)

			ts, err := time.Parse(time.RFC3339Nano, str)
			if err != nil {
				payload.Err = err
				continue
			}
			payload.Published = ts
		}
		if len(payload.ChannelId) > 0 && len(payload.VideoId) > 0 && seen {
			break
		}
	}
	if payload.Err != nil && (len(payload.ChannelId) == 0 || len(payload.VideoId) == 0) {
		payload.Err = fmt.Errorf("%s\n", "Failed to parse ID from XML data")
	}
	return payload
}

func isPublished(payload HookPayload) bool {
	now := time.Now().Unix()
	diff := now - payload.Published.Unix()

	if diff > MaxWait {
		return false
	}
	return true
}

func isXMLValid(payload HookPayload) bool {
	if payload.Err != nil {
		return true
	}
	if len(payload.ChannelId) == 0 || len(payload.VideoId) == 0 {
		return false
	}
	return true
}

func parseId(s string) string {
	start := 0
	end := len(s)

	for i := range s {
		start++
		if s[i] == '>' {
			break
		}
	}
	for i := start; i < len(s); i++ {
		end = i
		if s[i] == '<' {
			break
		}
	}
	return s[start:end]
}
