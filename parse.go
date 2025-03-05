package main

import "strings"

type HookPayload struct {
	channelId string
	videoId   string
}

func parseXML(data string) HookPayload {
	var payload HookPayload
	lines := strings.Split(data, "\n")

	for _, curr := range lines {
		s := strings.TrimSpace(curr)

		if strings.Contains(s, "yt:videoId") {
			payload.videoId = parseId(s)
			continue
		}
		if strings.Contains(s, "yt:channelId") {
			payload.channelId = parseId(s)
		}
		if len(payload.channelId) > 0 && len(payload.videoId) > 0 {
			break
		}
	}
	return payload
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
