package main

import "bot/philosophy/internal/database"

func prepareComments(ranked []RankedItem, stack []database.Cornucopium, budget int) []ReplyInfo {
	resp := []ReplyInfo{}
	capacity := budget / COMMENT_COST

	if len(stack) == 0 || capacity <= 0 {
		return resp
	}

	for i, comment := range ranked {
		if len(resp) >= capacity {
			return resp
		}
		idx := i % len(stack)
		curr := stack[idx]
		wisdom := constructWisdom(curr.Quote, curr.Author)

		reply := ReplyInfo{}
		payload := ReplyPayload{}

		payload.Snippet.ParentId = comment.Item.Id
		payload.Snippet.TextOriginal = wisdom

		reply.Payload = payload
		reply.Video_id = comment.VideoId
		reply.Quote_id = curr.ID

		resp = append(resp, reply)
	}
	return resp
}
