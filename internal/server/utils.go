package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
)

const TopicUrl = "https://www.youtube.com/feeds/videos.xml?channel_id="
const Route = "https://pubsubhubbub.appspot.com/subscribe"

func PostPubSub(channelId string, mode string) error {
	token := os.Getenv("BEARER")
	callback := os.Getenv("CALLBACK")
	topic := TopicUrl + channelId

	formData := url.Values{}
	formData.Set("hub.callback", callback)
	formData.Set("hub.topic", topic)
	formData.Set("hub.verify", "async")
	formData.Set("hub.mode", mode)
	formData.Set("hub.verify_token", token)

	client := http.Client{}
	payload := bytes.NewBufferString(formData.Encode())

	req, err := http.NewRequest(http.MethodPost, Route, payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 202 {
		return fmt.Errorf("%s\n", resp.Status)
	}
	return nil
}

func SuccessResp(w http.ResponseWriter, code int, payload interface{}) {
	resp, err := json.Marshal(payload)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(resp)
	return
}

func ErrorResp(w http.ResponseWriter, code int, msg string) {
	type ErrResp struct {
		Error string `json:"error"`
	}
	err_resp := ErrResp{Error: msg}
	resp, err := json.Marshal(err_resp)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(500)
		return
	}
	w.WriteHeader(code)
	w.Write(resp)
	return
}
