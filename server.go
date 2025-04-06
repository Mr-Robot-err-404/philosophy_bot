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
// channel frequency
// never repeat a quote on a channel

const Subscribe = "subscribe"
const Unsubscribe = "unsubscribe"
const MinWait = 2 * 60
const MaxWait = 10 * 60

func (cfg *Config) handlerDiogenes(w http.ResponseWriter, req *http.Request) {
	printBreak()
	fmt.Println("METHOD: ", req.Method)
	fmt.Println("QUERY:  ", req.URL.Query())

	comms := cfg.comms
	state := readServerState(comms.rd)

	if req.Method == http.MethodGet {
		challenge := req.URL.Query().Get("hub.challenge")
		w.Write([]byte(challenge))
		return
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		msg := "Failed to read XML body"
		comms.logs <- Log{Msg: msg}
		server.ErrorResp(w, http.StatusBadRequest, msg)
		return
	}
	defer req.Body.Close()
	evaluateXMLData(string(body), cfg.jobs, cfg.comms.logs, state.QuotaPoints, cfg.comms.points)

	rndId := uuid.New().String()
	fileName := rndId + ".xml"
	err = os.WriteFile("./tmp/xml/"+fileName, body, 0644)

	if err != nil {
		comms.logs <- Log{Err: fmt.Errorf("Failed to save XML file: %s", err.Error())}
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
	comms.logs <- Log{Msg: fmt.Sprintf("Created channel: %s", created.Handle)}

	server.SuccessResp(w, http.StatusCreated, "Created channel")
}

func (cfg *Config) handlerDeleteChannel(w http.ResponseWriter, req *http.Request) {
	comms := cfg.comms
	state := readServerState(comms.rd)

	token, err := auth.GetBearerToken(req.Header)

	if err != nil || token != state.Credentials.bearer {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	id := req.URL.Query().Get("id")

	if len(id) == 0 {
		server.ErrorResp(w, http.StatusBadRequest, "Channel ID required")
		return
	}
	channel, err := queries.FindChannel(ctx, id)

	if err != nil || len(channel.ID) == 0 {
		server.ErrorResp(w, http.StatusNotFound, "Channel ID not found")
		return
	}
	callback := "https://" + req.Host + "/diogenes/bowl"
	err = server.PostPubSub(channel.ID, Unsubscribe, callback, state.Credentials.bearer)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, err.Error())
		return
	}
	_, err = queries.DeleteChannel(ctx, channel.ID)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, err.Error())
		return
	}
	server.SuccessResp(w, http.StatusAccepted, fmt.Sprintf("Deleted channel: %s", channel.Handle))
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

	comms.logs <- Log{Msg: "New quote added"}

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
	mux.HandleFunc("DELETE /philosophy/channels", cfg.handlerDeleteChannel)
	mux.HandleFunc("POST /philosophy/quotes", cfg.handlerCreateQuote)
	mux.HandleFunc("GET /philosophy/logs", cfg.logHistoryHandler)
	mux.HandleFunc("GET /philosophy/points", cfg.QuotaPointsHandler)
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

	go stateManager(serverState, comms)
	go receiveJobs(cfg.jobs, results, cfg.comms.rd, cfg.comms.logs)
	go receiveTaskResults(results, cfg.comms.logs)
	go serverCronJob(cfg.comms.writeTkn, cfg.comms.logs)
	go renewSubscription(cfg.comms.logs, callback, credentials.bearer)

	subscribeToChannels(channels, callback, credentials.bearer, cfg.comms.logs)
	defer unsubscribeChannels(callback, credentials.bearer)

	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal(err)
	}
}
