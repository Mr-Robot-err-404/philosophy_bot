package main

import (
	"bot/philosophy/internal/auth"
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/google/uuid"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

type Config struct {
	credentials Credentials
}

type QuotePayload struct {
	Author     string `json:"author"`
	Quote      string `json:"quote"`
	Categories string `json:"categories,omitempty"`
}

func (cfg *Config) handlerDiogenes(w http.ResponseWriter, req *http.Request) {
	// HACK: authenticate -> make sure req is from youtube only
	// parse the data -> grab video_id, channel_id
	// post comment and save
}

func (cfg *Config) handlerCreateChannel(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != cfg.credentials.bearer {
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
		server.ErrorResp(w, http.StatusBadRequest, "Failed to get channel")
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

func (cfg *Config) handlerCreateQuote(w http.ResponseWriter, req *http.Request) {
	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != cfg.credentials.bearer {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)

	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	var payload QuotePayload
	err = json.Unmarshal(body, &payload)

	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	id := int64(uuid.New().ID())
	categories := "default"

	if len(payload.Categories) > 0 {
		categories = payload.Categories
	}
	params := database.CreateQuoteParams{ID: id, Quote: payload.Quote, Author: payload.Author, Categories: categories}
	_, err = queries.CreateQuote(ctx, params)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to save quote")
		return
	}
	server.SuccessResp(w, http.StatusCreated, "Created Quote")
}

func appHandler(prefix string, h http.Handler) http.Handler {
	return http.StripPrefix(prefix, h)
}

func startServer(credentials Credentials) {
	mux := http.NewServeMux()
	// srv := &http.Server{Handler: mux, Addr: ":6969"}
	cfg := Config{credentials: credentials}

	fileHnd := appHandler("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", fileHnd)

	mux.HandleFunc("POST /philosophy/channels", cfg.handlerCreateChannel)
	mux.HandleFunc("POST /philosophy/quotes", cfg.handlerCreateQuote)
	mux.HandleFunc("GET /diogenes/bowl", cfg.handlerDiogenes)

	fmt.Println("Philosophy Bot at your service")

	listener, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("App URL", listener.URL())

	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal(err)
	}
}
