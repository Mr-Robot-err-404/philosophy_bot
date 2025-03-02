package main

import (
	"bot/philosophy/internal/auth"
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"fmt"
	"log"
	"net/http"
)

// TODO: create a webhook for notifications
// endpoints:
//  |
//  -> add quotes & author
//  -> retrieve popular comments

type Config struct {
	secret      string
	credentials Credentials
}

func (cfg *Config) handlerCreateChannel(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != cfg.secret {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	tag := req.URL.Query().Get("tag")

	if len(tag) < 2 {
		server.ErrorResp(w, http.StatusBadRequest, "Invalid tag")
		return
	}
	channel, err := getChannel(tag, cfg.credentials.key)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to get channel")
		return
	}
	params := database.CreateChannelParams{ID: channel.Id, Title: channel.Snippet.Title, Handle: channel.Snippet.CustomUrl}
	_, err = queries.CreateChannel(ctx, params)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to create channel")
		return
	}
	server.SuccessResp(w, http.StatusCreated, "Created channel")
}

func appHandler(prefix string, h http.Handler) http.Handler {
	return http.StripPrefix(prefix, h)
}

func startServer(credentials Credentials) {
	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux, Addr: ":6969"}
	cfg := Config{credentials: credentials}

	fileHnd := appHandler("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", fileHnd)

	mux.HandleFunc("POST /philosophy/channels", cfg.handlerCreateChannel)

	fmt.Println("Philosophy Bot at your service")

	err := srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
