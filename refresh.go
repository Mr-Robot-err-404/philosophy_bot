package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

type OAuthResp struct {
	Access_token string
}

func refresh_token() (string, error) {
	config, err := extractGoogleConfig("client_secret.json")
	if err != nil {
		return "", err
	}
	refresh_token := os.Getenv("REFRESH_TOKEN")

	data := url.Values{}
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)
	data.Set("refresh_token", refresh_token)
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
