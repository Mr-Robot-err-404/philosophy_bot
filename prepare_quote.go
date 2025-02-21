package main

import "bot/philosophy/internal/database"

func prepareComments(ranked []RankedItem, stack []database.Cornucopium, quota int) []ReplyPayload {
	resp := []ReplyPayload{}
	capacity := (quota / COMMENT_COST) - 1

	for i, comment := range ranked {
		if len(resp) > capacity {
			return resp
		}
		idx := i % len(stack)
		curr := stack[idx]
		wisdom := constructWisdom(curr.Quote, curr.Author)

		reply := ReplyPayload{}
		reply.Snippet.ParentId = comment.Item.Id
		reply.Snippet.TextOriginal = wisdom
		resp = append(resp, reply)
	}
	return resp
}
