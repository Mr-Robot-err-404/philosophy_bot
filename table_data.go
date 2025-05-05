package main

import (
	"bot/philosophy/internal/database"
	"fmt"
	"time"
)

type TableCache struct {
	quotes   []database.Cornucopium
	comments []Comment
	replies  []Reply
	videos   []string
	channels []database.Channel
	login    database.Login
	quota    database.Quotum
}
type Reply struct {
	database.Reply
}
type Comment struct {
	database.Comment
}

func (C Comment) GetID() string {
	return C.ID
}
func (C Comment) GetLikes() int64 {
	return C.Likes
}
func (R Reply) GetID() string {
	return R.ID
}
func (R Reply) GetLikes() int64 {
	return R.Likes
}

type GenericComment interface {
	GetID() string
	GetLikes() int64
}

func getTableCache(access_token *string) (TableCache, error) {
	var cache TableCache

	quotes, err := queries.GetQuotes(ctx)
	if err != nil {
		return TableCache{}, err
	}
	videos, err := queries.GetVideos(ctx)
	if err != nil {
		return TableCache{}, err
	}
	quota, err := queries.GetQuota(ctx)
	if err != nil {
		return TableCache{}, err
	}
	login, err := queries.GetLoginDetails(ctx)
	if err != nil {
		return TableCache{}, err
	}
	replies, err := queries.GetReplies(ctx)
	if err != nil {
		return TableCache{}, err
	}
	comments, err := queries.GetComments(ctx)
	if err != nil {
		return TableCache{}, err
	}
	channels, err := queries.GetChannels(ctx)
	if err != nil {
		return TableCache{}, err
	}

	now := time.Now().Unix()
	elapsed := now - login.LastLogin.Unix()
	q_elapsed := now - quota.UpdatedAt.Unix()

	fmt.Println("time since last login: ", elapsed, "sec")

	if elapsed > 3000 {
		ts, err := renewSession(login.ID, access_token)
		if err != nil {
			return TableCache{}, err
		}
		fmt.Println("refreshed session")
		login.LastLogin = ts
	}
	if q_elapsed > int64(time.Hour*24) {
		updated, err := refresh_quota(quota.ID)
		if err != nil {
			return TableCache{}, err
		}
		fmt.Println("refreshed quota")
		quota.UpdatedAt = updated
	}
	cache.channels = channels
	cache.comments = convertComments(comments)
	cache.replies = convertReplies(replies)
	cache.login = login
	cache.quota = quota
	cache.quotes = quotes
	cache.videos = videos

	return cache, nil
}
func convertComments(comments []database.Comment) []Comment {
	resp := []Comment{}
	for _, curr := range comments {
		resp = append(resp, Comment{curr})
	}
	return resp
}
func convertReplies(replies []database.Reply) []Reply {
	resp := []Reply{}
	for _, curr := range replies {
		resp = append(resp, Reply{curr})
	}
	return resp
}
