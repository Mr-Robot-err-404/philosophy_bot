package main

import (
	"bot/philosophy/internal/database"
	"fmt"
)

func cleanup(v *int64, channel_id string, logs chan<- Log, seen chan<- SeenVid) {
	params := database.UpdateVideosSincePostParams{VideosSincePost: *v, ID: channel_id}
	err := updateSeen(params, seen)
	if err != nil {
		logs <- Log{Scope: "db", Msg: fmt.Sprintf("Failed to update video count for %s", channel_id), Err: err}
		return
	}
}
