package main

import (
	"bot/philosophy/internal/server"
	"net/http"
)

func (cfg *Config) handlerRefreshTkn(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tkn := req.URL.Query().Get("tkn")

	if len(tkn) < 100 {
		server.ErrorResp(w, 400, "Invalid token length")
		return
	}
	if err := saveRefreshToken(tkn); err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to persist refresh token")
		return
	}
	comms.refreshTkn <- WriteToken{token: tkn}
	server.SuccessResp(w, 202, "Updated refresh token")
}
