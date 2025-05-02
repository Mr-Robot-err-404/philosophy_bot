package main

import (
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (cfg *Config) handlerCreateChannel(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		return
	}
	tag := req.URL.Query().Get("tag")

	if len(tag) < 2 {
		server.ErrorResp(w, http.StatusBadRequest, "Invalid tag")
		return
	}
	channel, err := getChannel(tag, state.Credentials.key)

	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, "Failed to get channel")
		return
	}
	callback := "https://" + req.Host + "/diogenes/bowl"

	err = server.PostPubSub(channel.Id, Subscribe, callback, state.Credentials.bearer)
	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, err.Error())
		return
	}
	params := database.CreateChannelParams{
		ID:              channel.Id,
		Title:           channel.Snippet.Title,
		Handle:          channel.Snippet.CustomUrl,
		Frequency:       1,
		VideosSincePost: 0,
	}
	created := createChannel(params, cfg.dbComms.createChannel)

	if created.err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to create channel")
		return
	}
	comms.logs <- Log{Msg: fmt.Sprintf("Created channel: %s", created.channel.Handle)}

	server.SuccessResp(w, http.StatusCreated, "Created channel")
}

func (cfg *Config) handlerDeleteChannel(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		return
	}
	tag := req.URL.Query().Get("tag")

	if len(tag) == 0 {
		server.ErrorResp(w, http.StatusBadRequest, "Channel ID required")
		return
	}
	resp := readChannelByTag(tag, cfg.dbComms.rd.findTag)

	if resp.err != nil || len(resp.channel.ID) == 0 {
		server.ErrorResp(w, http.StatusNotFound, "Channel tag not found")
		return
	}
	channel := resp.channel
	callback := "https://" + req.Host + "/diogenes/bowl"
	err := server.PostPubSub(channel.ID, Unsubscribe, callback, state.Credentials.bearer)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, err.Error())
		return
	}
	err = simpleMan(channel.ID, cfg.dbComms.deleteChannel)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, err.Error())
		return
	}
	server.SuccessResp(w, http.StatusAccepted, fmt.Sprintf("Deleted channel: %s", channel.Handle))
}

func (cfg *Config) handlerUpdateFrequency(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		return
	}
	defer req.Body.Close()
	b, err := io.ReadAll(req.Body)

	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, err.Error())
		return
	}
	var payload FreqPayload
	err = json.Unmarshal(b, &payload)

	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(payload.Tag) == 0 || payload.Freq < 1 {
		server.ErrorResp(w, http.StatusBadRequest, "Tag required, freq must be > 0")
		return
	}
	resp := readChannelByTag(payload.Tag, cfg.dbComms.rd.findTag)

	if resp.err != nil {
		server.ErrorResp(w, http.StatusBadRequest, resp.err.Error())
		return
	}
	params := database.UpdateChannelFreqParams{Frequency: int64(payload.Freq), ID: resp.channel.ID}
	err = updateChannelFreq(params, cfg.dbComms.updateFreq)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, err.Error())
		return
	}
	server.SuccessResp(w, http.StatusCreated, fmt.Sprintf("Updated frequency for %s", payload.Tag))
}
