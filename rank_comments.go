package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type RankedItem struct {
	Item    ThreadItem
	Score   int
	VideoId string
}

func rankComments(comments []ThreadItem, video_id string) []RankedItem {
	best := []RankedItem{}

	first := RankedItem{VideoId: video_id}
	second := RankedItem{VideoId: video_id}
	third := RankedItem{VideoId: video_id}

	for i := range comments {
		item := comments[i]
		replies := item.Snippet.TotalReplyCount
		likes := item.Snippet.TopLevelComment.Snippet.LikeCount

		if replies > 2 || likes < 100 {
			continue
		}
		score := likes
		penalty := replies * 50
		score -= penalty

		if score > first.Score {
			third.Item = second.Item
			third.Score = second.Score
			second.Item = first.Item
			second.Score = first.Score
			first.Item = item
			first.Score = score
			continue
		}
		if score > second.Score {
			third.Item = second.Item
			third.Score = second.Score
			second.Item = item
			second.Score = score
			continue
		}
		if score > third.Score {
			third.Item = item
			third.Score = score
		}
	}
	if first.Score > 0 {
		best = append(best, first)
	}
	if second.Score > 0 {
		best = append(best, second)
	}
	if third.Score > 0 {
		best = append(best, third)
	}
	return best
}

func simulateRanking() error {
	data, err := os.ReadFile("./public/marmot.json")
	if err != nil {
		return err
	}
	var comments []ThreadItem
	err = json.Unmarshal(data, &comments)
	if err != nil {
		return err
	}
	ranked := rankComments(comments, "some_id")

	for _, curr := range ranked {
		fmt.Println("SCORE -> ", curr.Score)
		fmt.Println("LIKES -> ", curr.Item.Snippet.TopLevelComment.Snippet.LikeCount)
		fmt.Println("REPLY -> ", curr.Item.Snippet.TotalReplyCount)
		fmt.Println("TEXT  -> ", curr.Item.Snippet.TopLevelComment.Snippet.TextDisplay)
		fmt.Println("----------")
	}
	return nil
}
