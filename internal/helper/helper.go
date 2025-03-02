package helper

import "fmt"

const CommentStats = "https://youtube.googleapis.com/youtube/v3/comments?part=snippet&id="

func BuildStatsUrl(comments []string, key string) (string, error) {
	url := CommentStats

	if len(comments) == 0 {
		return url, fmt.Errorf("%s\n", "Invalid comment length. Require: 1 -> 100")
	}
	url += comments[0]

	for i := 1; i < len(comments); i++ {
		id := comments[i]
		url += "," + id
	}
	url += "&key=" + key
	return url, nil
}
