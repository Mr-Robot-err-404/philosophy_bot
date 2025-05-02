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
	likes       map[string]int
}

type Config struct {
	jobs    chan Worker
	comms   Comms
	dbComms DbComms
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
	deleteChannel chan SimpleMan
	saveVid       chan SimpleMan
	saveReply     chan SaveReply
	saveComment   chan database.CreateCommentParams
	saveUsage     chan Usage
	saveQuota     chan int
	seenVid       chan SeenVid
	createChannel chan CreateChannel
	wisdom        chan Wisdom
	resetQuota    chan bool
	updateFreq    chan Freq
	rd            DbReadComms
}
type SeenVid struct {
	params database.UpdateVideosSincePostParams
	resp   chan error
}
type CreateChannel struct {
	params database.CreateChannelParams
	resp   chan CreateResp
}
type SimpleMan struct {
	id   string
	resp chan error
}
type SaveReply struct {
	params database.StoreReplyParams
	resp   chan ReplyResp
}
type ReplyResp struct {
	reply database.Reply
	err   error
}
type CreateResp struct {
	err     error
	channel database.Channel
}
type Wisdom struct {
	epiphany database.CreateQuoteParams
	resp     chan WisdomResp
}
type WisdomResp struct {
	err   error
	quote database.Cornucopium
}

type Freq struct {
	params database.UpdateChannelFreqParams
	resp   chan error
}

type Usage struct {
	channelId string
	quoteId   int64
}

// TODO: stats cron -> done?

// HACK:
// "no u" channel owner listener
// user interactions

// TEST: -> endpoints
// update channel freq
// delete channel
// log history
// get quota
// stats

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
	evaluateXMLData(string(body), state.QuotaPoints, cfg)

	rndId := uuid.New().String()
	fileName := rndId + ".xml"
	err = os.WriteFile("./tmp/xml/"+fileName, body, 0644)

	if err != nil {
		comms.logs <- Log{Err: fmt.Errorf("Failed to save XML file: %s", err.Error())}
	}
	server.SuccessResp(w, 200, "Accepted")
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
	resp := suddenEpiphany(params, cfg.dbComms.wisdom)

	if resp.err != nil {
		server.ErrorResp(w, http.StatusInternalServerError, "Failed to save quote")
		return
	}
	wr := WriteQuote{quote: resp.quote, resp: make(chan bool)}
	comms.writeWisdom <- wr
	<-wr.resp

	comms.logs <- Log{Msg: "New quote added"}
	server.SuccessResp(w, http.StatusCreated, "Created Quote")
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
	cfg.dbComms = dbComms

	fileHnd := appHandler("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", fileHnd)

	mux.HandleFunc("POST /philosophy/channels", cfg.handlerCreateChannel)
	mux.HandleFunc("DELETE /philosophy/channels", cfg.handlerDeleteChannel)
	mux.HandleFunc("UPDATE /philosophy/channels", cfg.handlerUpdateFrequency)
	mux.HandleFunc("POST /philosophy/quotes", cfg.handlerCreateQuote)
	mux.HandleFunc("GET /philosophy/logs", cfg.logHistoryHandler)
	mux.HandleFunc("GET /philosophy/points", cfg.QuotaPointsHandler)
	mux.HandleFunc("GET /philosophy/stats", cfg.handlerStats)
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

	go stateManager(serverState, &cfg.comms, &dbComms)
	go dbManager(&dbComms, cfg.comms.logs)

	go receiveJobs(cfg.jobs, results, &cfg.comms, &dbComms)
	go receiveTaskResults(results, cfg.comms.logs, &dbComms)

	go serverCronJob(&cfg.comms, &cfg.dbComms)
	go renewSubscription(cfg.comms.logs, callback, credentials.bearer, &cfg.dbComms)

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
	comms.writeSeen = make(chan string)
	comms.logs = make(chan Log)
	comms.points = make(chan UpdateQuotaPoints)

	rdComms := DbReadComms{}
	rdComms.findTag = make(chan FindTag)
	rdComms.get = make(chan GetChannel)
	rdComms.getAll = make(chan GetAll)
	rdComms.unused = make(chan GetUnused)
	rdComms.popularComments = make(chan PopularComments)
	rdComms.popularReplies = make(chan PopularReplies)
	dbComms.rd = rdComms

	dbComms.saveComment = make(chan database.CreateCommentParams)
	dbComms.saveQuota = make(chan int)
	dbComms.saveUsage = make(chan Usage)

	dbComms.createChannel = make(chan CreateChannel)
	dbComms.deleteChannel = make(chan SimpleMan)

	dbComms.saveVid = make(chan SimpleMan, 50)
	dbComms.saveReply = make(chan SaveReply, 50)
	dbComms.seenVid = make(chan SeenVid)

	dbComms.wisdom = make(chan Wisdom)
	dbComms.updateFreq = make(chan Freq)
	dbComms.resetQuota = make(chan bool)
}
