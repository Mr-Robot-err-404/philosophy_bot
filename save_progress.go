package main

import (
	"bot/philosophy/internal/database"
	"fmt"
)

func saveProgress(replies []WiseReply) {
	vids := unique_vids(replies)
	err_resp := []error{}

	for _, id := range vids {
		video, err := queries.SaveVideo(ctx, id)
		if err != nil {
			err_resp = append(err_resp, err)
			continue
		}
		fmt.Println("Saved vid -> ", video)
	}
	if len(err_resp) > 0 {
		fmt.Println("ERR SAVING VID: ")
	}
	logErrors(err_resp)
	err_resp = []error{}

	for _, item := range replies {
		params := database.StoreReplyParams{ID: item.Reply.Id, Likes: 0, QuoteID: item.Quote_id, VideoID: item.Video_id}
		_, err := queries.StoreReply(ctx, params)
		if err != nil {
			err_resp = append(err_resp, err)
			continue
		}
		fmt.Println("Saved reply -> ", item.Reply.Id)
	}
	if len(err_resp) > 0 {
		fmt.Println("ERR STORING REPLY: ")
	}
	logErrors(err_resp)
}

func unique_vids(replies []WiseReply) []string {
	vids := []string{}
	seen := make(map[string]string)

	for _, reply := range replies {
		_, exists := seen[reply.Video_id]
		if exists {
			continue
		}
		vids = append(vids, reply.Video_id)
	}
	return vids
}
