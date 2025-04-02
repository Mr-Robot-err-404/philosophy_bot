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
	"os"
	"time"

	"github.com/google/uuid"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
)

type Config struct {
	jobs  chan Worker
	comms Comms
}

type ServerState struct {
	Credentials Credentials
	Quotes      []database.Cornucopium
	LogHistory  []Log
	QuotaPoints int
}

type QuotePayload struct {
	Author     string `json:"author"`
	Quote      string `json:"quote"`
	Categories string `json:"categories,omitempty"`
}
type Comms struct {
	rd          chan ReadReq
	writeWisdom chan WriteQuote
	writeTkn    chan WriteAccessToken
	logs        chan Log
	points      chan UpdateQuotaPoints
}

// TODO: track quota
//  -> Calculate quota margins before batch update
//  -> Combine quota cron job with trending cron

// TODO:
// pubsub cron job - refresh subscriptions
// channel frequency

const Subscribe = "subscribe"
const Unsubscribe = "unsubscribe"
const MinWait = 2 * 60
const MaxWait = 10 * 60

func (cfg *Config) handlerDiogenes(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	if req.Method == http.MethodGet {
		challenge := req.URL.Query().Get("hub.challenge")
		w.Write([]byte(challenge))
		return
	}
	if req.Method != http.MethodPost {
		comms.logs <- Log{msg: fmt.Sprintf("Invalid method: %s\n", req.Method), ts: time.Now()}
		server.ErrorResp(w, http.StatusMethodNotAllowed, "Invalid method")
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		msg := "Failed to read XML body"

		comms.logs <- Log{msg: msg, ts: time.Now()}
		server.ErrorResp(w, http.StatusBadRequest, msg)
		return
	}
	defer req.Body.Close()
	evaluateXMLData(string(body), cfg.jobs, cfg.comms.logs, state.QuotaPoints, cfg.comms.points)

	rndId := uuid.New().String()
	fileName := rndId + ".xml"
	err = os.WriteFile("./tmp/xml/"+fileName, body, 0644)

	if err != nil {
		comms.logs <- Log{err: fmt.Errorf("Failed to save XML file: %s\n", err.Error()), ts: time.Now()}
	}
	server.SuccessResp(w, 200, "Accepted")
}

func (cfg *Config) handlerCreateChannel(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != state.Credentials.bearer {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
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
	created, err := queries.CreateChannel(ctx, params)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to create channel")
		return
	}
	comms.logs <- Log{msg: fmt.Sprintf("Created channel: %s\n", created.Handle)}

	server.SuccessResp(w, http.StatusCreated, "Created channel")
}

func (cfg *Config) handlerCreateQuote(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != state.Credentials.bearer {
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
	quote, err := queries.CreateQuote(ctx, params)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to save quote")
		return
	}
	wr := WriteQuote{quote: quote, resp: make(chan bool)}
	comms.writeWisdom <- wr
	<-wr.resp

	comms.logs <- Log{msg: "New quote added", ts: time.Now()}

	server.SuccessResp(w, http.StatusCreated, "Created Quote")
}

func (cfg *Config) logHistoryHandler(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != state.Credentials.bearer {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	server.SuccessResp(w, http.StatusOK, recentLogs(state.LogHistory))
}

func (cfg *Config) QuotaPointsHandler(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)
	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != state.Credentials.bearer {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	server.SuccessResp(w, http.StatusOK, state.QuotaPoints)
}

func appHandler(prefix string, h http.Handler) http.Handler {
	return http.StripPrefix(prefix, h)
}

func startServer(credentials Credentials, quotes []database.Cornucopium, channels []database.Channel) {
	mux := http.NewServeMux()

	cfg := Config{}
	comms := Comms{}
	serverState := ServerState{Credentials: credentials, Quotes: quotes, QuotaPoints: 10000}

	results := make(chan TaskResult)
	comms.rd = make(chan ReadReq)
	comms.writeWisdom = make(chan WriteQuote)
	comms.writeTkn = make(chan WriteAccessToken)
	comms.logs = make(chan Log)
	comms.points = make(chan UpdateQuotaPoints)

	cfg.jobs = make(chan Worker)
	cfg.comms = comms

	fileHnd := appHandler("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", fileHnd)

	mux.HandleFunc("POST /philosophy/channels", cfg.handlerCreateChannel)
	mux.HandleFunc("POST /philosophy/quotes", cfg.handlerCreateQuote)
	mux.HandleFunc("GET /philosophy/logs", cfg.logHistoryHandler)
	mux.HandleFunc("/diogenes/bowl", cfg.handlerDiogenes)

	listener, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("App URL", listener.URL())

	callback := listener.URL() + "/diogenes/bowl"
	err = subscribeToChannels(channels, callback, credentials.bearer)

	if err != nil {
		log.Fatal(err)
	}
	go stateManager(serverState, comms)
	go receiveJobs(cfg.jobs, results, cfg.comms.rd, cfg.comms.logs)
	go receiveTaskResults(results, cfg.comms.logs)
	go serverCronJob(cfg.comms.writeTkn, cfg.comms.logs)

	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal(err)
	}
}
