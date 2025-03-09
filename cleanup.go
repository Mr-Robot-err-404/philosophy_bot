package main

import (
	"bot/philosophy/internal/database"
	"fmt"
)

func cleanup(v *int64, channel_id string) {
	params := database.UpdateVideosSincePostParams{VideosSincePost: *v, ID: channel_id}
	_, err := queries.UpdateVideosSincePost(ctx, params)

	if err != nil {
		fmt.Println(err)
		return
	}
}
