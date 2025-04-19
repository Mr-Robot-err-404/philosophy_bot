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

type Startup struct {
	credentials Credentials
	quotes      []database.Cornucopium
	channels    []database.Channel
	seen        map[string]bool
}

type Config struct {
	jobs  chan Worker
	comms Comms
}

type ServerState struct {
	Credentials Credentials
	Quotes      []database.Cornucopium
	LogHistory  []Log
	QuotaPoints int
	Seen        map[string]bool
}

type QuotePayload struct {
	Author     string `json:"author"`
	Quote      string `json:"quote"`
	Categories string `json:"categories,omitempty"`
}
type FreqPayload struct {
	Tag  string `json:"tag"`
	Freq int    `json:"freq"`
}
type Comms struct {
	rd          chan ReadReq
	writeWisdom chan WriteQuote
	writeTkn    chan WriteAccessToken
	writeSeen   chan string
	logs        chan Log
	points      chan UpdateQuotaPoints
}
type DbComms struct {
	saveVid     chan string
	saveComment chan database.CreateCommentParams
	saveUsage   chan Usage
	saveQuota   chan int
	resetQuota  chan bool
}
type Usage struct {
	channelId string
	quoteId   int64
}

// TODO:
// log history response
// stats endpoints:
// -> most popular comments

// HACK:
// "no u" channel owner listener
// user interactions

// TEST: -> endpoints
// update channel freq
// delete channel
// log history
// get quota

const Subscribe = "subscribe"
const Unsubscribe = "unsubscribe"
const MinWait = 2 * 60
const MaxWait = 10 * 60

func checkTkn(header http.Header, w http.ResponseWriter, bearer string) bool {
	token, err := auth.GetBearerToken(header)
	if err != nil || token != bearer {
		server.ErrorResp(w, http.StatusUnauthorized, "Unauthorized")
		return false
	}
	return true
}

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
	channel, err := queries.FindTag(ctx, payload.Tag)

	if err != nil {
		server.ErrorResp(w, http.StatusBadRequest, err.Error())
		return
	}
	params := database.UpdateChannelFreqParams{Frequency: int64(payload.Freq), ID: channel.ID}
	_, err = queries.UpdateChannelFreq(ctx, params)

	if err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, err.Error())
		return
	}
	server.SuccessResp(w, http.StatusCreated, fmt.Sprintf("Updated frequency for %s", payload.Tag))
}

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

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
		return
	}
	tag := req.URL.Query().Get("tag")

	if len(tag) == 0 {
		server.ErrorResp(w, http.StatusBadRequest, "Channel ID required")
		return
	}
	channel, err := queries.FindTag(ctx, tag)

	if err != nil || len(channel.ID) == 0 {
		server.ErrorResp(w, http.StatusNotFound, "Channel tag not found")
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

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
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

	if !checkTkn(req.Header, w, state.Credentials.bearer) {
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

func appHandler(prefix string, h http.Handler) http.Handler {
	return http.StripPrefix(prefix, h)
}

func startServer(startup Startup) {
	mux := http.NewServeMux()
	credentials, quotes := startup.credentials, startup.quotes
	channels, seen := startup.channels, startup.seen

	cfg := Config{}
	comms := Comms{}
	dbComms := DbComms{}
	serverState := ServerState{Credentials: credentials, Quotes: quotes, QuotaPoints: 10000, Seen: seen}

	initComms(&comms, &dbComms)

	results := make(chan TaskResult)
	cfg.jobs = make(chan Worker)
	cfg.comms = comms

	fileHnd := appHandler("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", fileHnd)

	mux.HandleFunc("POST /philosophy/channels", cfg.handlerCreateChannel)
	mux.HandleFunc("DELETE /philosophy/channels", cfg.handlerDeleteChannel)
	mux.HandleFunc("UPDATE /philosophy/channels", cfg.handlerUpdateFrequency)
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

	go stateManager(serverState, &comms, &dbComms)
	go dbManager(&dbComms, comms.logs)

	go receiveJobs(cfg.jobs, results, &comms, &dbComms)
	go receiveTaskResults(results, cfg.comms.logs, &dbComms)

	go serverCronJob(&cfg.comms)
	go renewSubscription(cfg.comms.logs, callback, credentials.bearer)

	subscribeToChannels(channels, callback, credentials.bearer, cfg.comms.logs)
	defer unsubscribeChannels(callback, credentials.bearer)

	err = http.Serve(listener, mux)
	if err != nil {
		log.Fatal(err)
	}
}

func initComms(comms *Comms, dbComms *DbComms) {
	comms.rd = make(chan ReadReq)
	comms.writeWisdom = make(chan WriteQuote)
	comms.writeTkn = make(chan WriteAccessToken)
	comms.logs = make(chan Log)
	comms.points = make(chan UpdateQuotaPoints)
	comms.writeSeen = make(chan string)

	dbComms.saveVid = make(chan string)
	dbComms.saveComment = make(chan database.CreateCommentParams)
	dbComms.saveQuota = make(chan int)
	dbComms.resetQuota = make(chan bool)
	dbComms.saveUsage = make(chan Usage)
}
