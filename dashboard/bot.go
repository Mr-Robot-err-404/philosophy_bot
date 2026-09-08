package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type Job struct {
	Name      string `json:"name"`
	Interval  int    `json:"interval_seconds"`
	NextRun   string `json:"next_run"`
	Remaining int    `json:"seconds_remaining"`
	Stopped   bool   `json:"stopped"`
}

type BotClient struct {
	addr   string
	bearer string
	http   *http.Client
}

func newBotClient() *BotClient {
	addr := os.Getenv("BOT_ADDR")
	if addr == "" {
		addr = "127.0.0.1:49399"
	}
	return &BotClient{
		addr:   addr,
		bearer: os.Getenv("BEARER"),
		http:   &http.Client{Timeout: 2 * time.Second},
	}
}

func (c *BotClient) schedule() ([]Job, error) {
	url := fmt.Sprintf("http://%s/kant/schedule", c.addr)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.bearer)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bot returned %s", resp.Status)
	}
	var jobs []Job
	if err := json.NewDecoder(resp.Body).Decode(&jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}
