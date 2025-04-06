package main

import (
	"bot/philosophy/internal/database"
	"fmt"
)

func saveProgress(replies []WiseReply) {
	if len(replies) == 0 {
		return
	}
	vids := unique_vids(replies)
	err_resp := []error{}

	printBreak()
	for _, id := range vids {
		video, err := queries.SaveVideo(ctx, id)
		if err != nil {
			err_resp = append(err_resp, err)
			continue
		}
		fmt.Println(video)
	}
	if len(err_resp) > 0 {
		fmt.Println("ERR SAVING VID: ")
	}
	printBreak()

	logErrors(err_resp)
	err_resp = []error{}

	for _, item := range replies {
		params := database.StoreReplyParams{ID: item.Reply.Id, Likes: 0, QuoteID: item.Quote_id, VideoID: item.Video_id}
		_, err := queries.StoreReply(ctx, params)
		if err != nil {
			err_resp = append(err_resp, err)
			continue
		}
		fmt.Println(item.Reply.Id)
	}
	if len(err_resp) > 0 {
		printBreak()
		fmt.Println("ERR STORING REPLY: ")
	}
	logErrors(err_resp)
}

func updateLikes(stats []UpdatedStats) ([]database.Reply, []error) {
	err_resp := []error{}
	success := []database.Reply{}

	for _, curr := range stats {
		params := database.UpdateLikesParams{Likes: int64(curr.likes), ID: curr.id}
		reply, err := queries.UpdateLikes(ctx, params)

		if err != nil {
			err_resp = append(err_resp, err)
			continue
		}
		success = append(success, reply)
	}
	return success, err_resp
}

func unique_vids(replies []WiseReply) []string {
	vids := []string{}
	seen := make(map[string]bool)

	for _, reply := range replies {
		_, exists := seen[reply.Video_id]
		if exists {
			continue
		}
		vids = append(vids, reply.Video_id)
		seen[reply.Video_id] = true
	}
	return vids
}
