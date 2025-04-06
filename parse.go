package main

import (
	"fmt"
	"strings"
	"time"
)

type HookPayload struct {
	ChannelId string
	VideoId   string
	Published TimeData
	Updated   TimeData
	Err       error
}

type TimeData struct {
	Exists bool
	Time   time.Time
}

func parseXML(data string) HookPayload {
	var payload HookPayload
	lines := strings.Split(data, "\n")
	entry := false

	for _, curr := range lines {
		if payload.Err != nil {
			break
		}
		s := strings.TrimSpace(curr)

		if strings.Contains(s, "<entry>") {
			entry = true
		}
		if entry != true {
			continue
		}
		if strings.Contains(s, "</entry>") {
			break
		}
		if strings.Contains(s, "<yt:videoId>") {
			payload.VideoId = parseId(s, "<yt:videoId>")
		}
		if strings.Contains(s, "<yt:channelId>") {
			payload.ChannelId = parseId(s, "<yt:channelId>")
		}
		if strings.Contains(s, "<published>") {
			str := parseId(s, "<published>")

			ts, err := time.Parse(time.RFC3339Nano, str)
			if err != nil {
				payload.Err = err
				continue
			}
			payload.Published = TimeData{Time: ts, Exists: true}
		}
		if strings.Contains(s, "<updated>") {
			str := parseId(s, "<updated>")

			ts, err := time.Parse(time.RFC3339Nano, str)
			if err != nil {
				payload.Err = err
				continue
			}
			payload.Updated = TimeData{Time: ts, Exists: true}
		}
	}
	payload.Err = validateXMLData(payload)
	return payload
}

func validateXMLData(payload HookPayload) error {
	if payload.Err != nil {
		return payload.Err
	}
	if len(payload.ChannelId) == 0 {
		return fmt.Errorf("%s", "ChannelId not found")
	}
	if len(payload.VideoId) == 0 {
		return fmt.Errorf("%s", " not found")
	}
	if !payload.Published.Exists {
		return fmt.Errorf("%s", "Published date does not exist in XML data")
	}
	if payload.Updated.Exists {
		published := payload.Published.Time
		updated := payload.Updated.Time
		return fmt.Errorf("Video already published. Published: %v, Updated: %v", published, updated)
	}
	return nil
}

func parseId(s string, tag string) string {
	start := startIdx(s, tag)
	if start == -1 {
		return ""
	}
	end := start

	for i := start; i < len(s); i++ {
		end = i
		if s[i] == '<' {
			break
		}
	}
	return s[start:end]
}

func startIdx(str string, subStr string) int {
	for i := range str {
		end := i + len(subStr)

		if end > len(str) {
			return -1
		}
		tmp := str[i:end]

		if tmp != subStr {
			continue
		}
		return end
	}
	return -1
}
