package main

import (
	"bot/philosophy/internal/database"
)

func cleanup(v *int64, channel_id string, logs chan<- Log, seen chan<- SeenVid) {
	params := database.UpdateVideosSincePostParams{VideosSincePost: *v, ID: channel_id}
	err := updateSeen(params, seen)
	if err != nil {
		logs <- Log{Err: err}
		return
	}
}
