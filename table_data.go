package main

import (
	"bot/philosophy/internal/database"
	"fmt"
	"time"
)

type TableCache struct {
	quotes []database.Cornucopium
	videos []string
	login  database.Login
	quota  database.Quotum
}

func getTableCache() (TableCache, error) {
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
	now := time.Now().Unix()
	elapsed := now - login.LastLogin.Unix()
	q_elapsed := now - quota.UpdatedAt.Unix()

	fmt.Println("time since last login: ", elapsed, "sec")

	if elapsed > 3000 {
		ts, err := refreshSession(login.ID)
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
	return TableCache{quotes: quotes, videos: videos, quota: quota, login: login}, nil
}
