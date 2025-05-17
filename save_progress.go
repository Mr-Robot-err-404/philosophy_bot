package main

import (
	"bot/philosophy/internal/database"
	"fmt"
)

func saveProgress(replies []WiseReply, dbComms *DbComms, logs chan<- Log, seen chan<- string) {
	if len(replies) == 0 {
		return
	}
	vids := unique_vids(replies)
	msg := "Saved vids: "

	for _, id := range vids {
		seen <- id
		err := simpleMan(id, dbComms.saveVid)
		if err != nil {
			logs <- Log{Err: err}
			continue
		}
		msg += fmt.Sprintf("%s | ", id)
	}
	logs <- Log{Msg: msg}
	msg = "Saved replies: "

	for _, item := range replies {
		params := database.StoreReplyParams{ID: item.Reply.Id, Likes: 0, QuoteID: item.Quote_id, VideoID: item.Video_id}
		resp := saveReply(params, dbComms.saveReply)

		if resp.err != nil {
			logs <- Log{Err: resp.err}
			continue
		}
		msg += fmt.Sprintf("%s | ", resp.reply.ID)
	}
	logs <- Log{Msg: msg}
}

func storeProgress(replies []WiseReply) {
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

func storeLikes(stats []UpdatedStats, logs chan<- Log, table string) {
	c := 0

	for _, curr := range stats {
		var err error

		if table == "replies" {
			params := database.UpdateReplyLikesParams{Likes: int64(curr.likes), ID: curr.id}
			_, err = queries.UpdateReplyLikes(ctx, params)
		} else {
			params := database.UpdateCommentLikesParams{Likes: int64(curr.likes), ID: curr.id}
			_, err = queries.UpdateCommentLikes(ctx, params)
		}

		if err != nil {
			logs <- Log{Err: err}
			continue
		}
		c++
	}
	logs <- Log{Msg: fmt.Sprintf("Updated %d likes for %s", c, table)}
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
