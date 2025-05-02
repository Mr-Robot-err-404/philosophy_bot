package main

import (
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"net/http"
)

type StatsPayload struct {
	Comments []database.GetPopularCommentsRow `json:"comments"`
	Replies  []database.GetPopularRepliesRow  `json:"replies"`
}

func (cfg *Config) handlerStats(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		return
	}
	resp := getPopularComments(cfg.dbComms.rd.popularComments)

	if resp.err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, resp.err.Error())
		return
	}
	response := getPopularReplies(cfg.dbComms.rd.popularReplies)

	if response.err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, response.err.Error())
		return
	}
	payload := StatsPayload{Comments: resp.comments, Replies: response.replies}
	server.SuccessResp(w, http.StatusOK, payload)
}

func (cfg *Config) logHistoryHandler(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorizaed")
		return
	}
	server.SuccessResp(w, http.StatusOK, recentLogs(state.LogHistory))
}

func (cfg *Config) QuotaPointsHandler(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		return
	}
	server.SuccessResp(w, http.StatusOK, state.QuotaPoints)
}
