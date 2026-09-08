package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/joho/godotenv"
)

type OAuthResp struct {
	Access_token string
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

func refresh_quota(id string) (time.Time, error) {
	quota, err := queries.RefreshQuota(ctx)
	if err != nil {
		return time.Time{}, err
	}
	return quota.UpdatedAt, nil
}

func renewAccessToken(access_token string) error {
	env, err := getEnvMap()
	if err != nil {
		return err
	}
	env["ACCESS_TOKEN"] = access_token
	err = godotenv.Write(env, "./.env")
	if err != nil {
		return err
	}
	return nil
}

func getEnvMap() (map[string]string, error) {
	var env_map map[string]string
	env_map, err := godotenv.Read()

	if err != nil {
		return env_map, err
	}
	return env_map, nil
}
