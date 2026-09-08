package main

import (
	"bot/philosophy/internal/database"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type OAuthResp struct {
	Access_token  string `json:"access_token"`
	Refresh_token string `json:"refresh_token"`
}

func refresh_token(tkn string) (string, error) {
	config, err := extractGoogleConfig("client_secret.json")
	if err != nil {
		return "", err
	}
	data := url.Values{}
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("refresh_token", tkn)
	data.Set("grant_type", "refresh_token")

	route := "https://www.googleapis.com/oauth2/v4/token"
	resp, err := http.PostForm(route, data)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		err := fmt.Errorf("%s\n", resp.Status)
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return "", err
	}
	var s OAuthResp
	err = json.Unmarshal(body, &s)
	if err != nil {
		return "", err
	}
	if s.Refresh_token != "" && s.Refresh_token != tkn {
		if err := saveRefreshToken(s.Refresh_token); err != nil {
			return "", err
		}
	}
	return s.Access_token, nil
}

func refreshAndRenewToken(tkn *string, refresh_tkn string) error {
	access_token, err := refresh_token(refresh_tkn)
	if err != nil {
		return err
	}
	*tkn = access_token
	err = renewAccessToken(access_token)
	return err
}

func renewSession(id string, tkn *string, refresh_tkn string) (time.Time, error) {
	err := refreshAndRenewToken(tkn, refresh_tkn)
	if err != nil {
		return time.Time{}, err
	}
	login, err := queries.UpdateLogin(ctx, id)
	if err != nil {
		return time.Time{}, err
	}
	return login.LastLogin, nil
}

func refresh_quota(id string) (database.Quotum, error) {
	quota, err := queries.RefreshQuota(ctx)
	if err != nil {
		return database.Quotum{}, err
	}
	return quota, nil
}

func renewAccessToken(access_token string) error {
	return saveAccessToken(access_token)
}
