package main

import (
	"bot/philosophy/email"
	"bot/philosophy/internal/auth"
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/server"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"golang.org/x/oauth2"
)

type Startup struct {
	credentials   Credentials
	quotes        []database.Cornucopium
	channels      []database.Channel
	seen          map[string]bool
	likes         map[string]int
	cache         TableCache
	google_config *oauth2.Config
	email_payload email.Payload
	addr          string
}

type Config struct {
	comms   Comms
	dbComms DbComms
	oauth   *oauth2.Config
	email   email.Payload
}

type ServerState struct {
	Credentials Credentials
	Quotes      []database.Cornucopium
	LogHistory  []Log
	Schedule    map[string]time.Time
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
	writeSeen   chan string
	writeTkn    chan WriteToken
	refreshTkn  chan WriteToken
	logs        chan Log
	points      chan UpdateQuotaPoints
	schedule    chan ScheduleTick
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
	tasks         DbTaskComms
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

// TODO:
// create binary
// host on raspberry pi

// HACK:
// "no u" channel owner listener
// user interactions

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
	signals, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	done := make(chan struct{})
	mux := http.NewServeMux()
	credentials, quotes := startup.credentials, startup.quotes
	channels, seen := startup.channels, startup.seen

	cfg := Config{}
	comms := Comms{}
	dbComms := DbComms{}
	schedule, notes := resumeSchedule(time.Now())

	serverState := ServerState{
		Credentials: credentials,
		Quotes:      quotes,
		QuotaPoints: int(startup.cache.quota.Quota),
		Seen:        seen,
		LogHistory:  make([]Log, 0, MaxLogHistory),
		Schedule:    schedule,
	}

	initComms(&comms, &dbComms)

	cfg.comms = comms
	cfg.dbComms = dbComms
	cfg.email = startup.email_payload
	cfg.oauth = startup.google_config

	fileHnd := appHandler("/app/", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", fileHnd)

	mux.HandleFunc("POST /socrates/channels", cfg.handlerCreateChannel)
	mux.HandleFunc("DELETE /socrates/channels", cfg.handlerDeleteChannel)
	mux.HandleFunc("UPDATE /socrates/channels", cfg.handlerUpdateFrequency)
	mux.HandleFunc("POST /kafka/quotes", cfg.handlerCreateQuote)
	mux.HandleFunc("POST /kafka/refresh", cfg.handlerRefreshTkn)
	mux.HandleFunc("GET /kant/logs", cfg.logHistoryHandler)
	mux.HandleFunc("GET /kant/points", cfg.QuotaPointsHandler)
	mux.HandleFunc("GET /kant/schedule", cfg.handlerSchedule)
	mux.HandleFunc("GET /kant/stats", cfg.handlerStats)
	mux.HandleFunc("/diogenes/bowl", cfg.handlerDiogenes)

	listener, err := ngrok.Listen(ctx,
		config.HTTPEndpoint(),
		ngrok.WithAuthtokenFromEnv(),
	)
	if err != nil {
		log.Fatal(err)
	}
	callback := listener.URL() + "/diogenes/bowl"

	go stateManager(serverState, &cfg.comms, &dbComms)
	go dbManager(&dbComms, cfg.comms.logs)

	go taskWorker(signals, &cfg.comms, &dbComms)

	go serverCronJob(&cfg.comms, &cfg.dbComms, cfg.email, serverState.Schedule, done)
	go renewSubscription(cfg.comms.logs, callback, credentials.bearer, &cfg.dbComms)

	subscribeToChannels(channels, callback, credentials.bearer, cfg.comms.logs)

	printBanner([]string{
		"philosophy bot",
		fmt.Sprintf("quotes %d | channels %d | seen %d", len(quotes), len(channels), len(seen)),
		fmt.Sprintf("quota %d | local %s", serverState.QuotaPoints, startup.addr),
		listener.URL(),
	})
	comms.logs <- Log{Scope: "http", Msg: fmt.Sprintf("Public callback -> %s", callback)}

	for _, note := range notes {
		comms.logs <- Log{Scope: "cron", Msg: note}
	}

	local := &http.Server{
		Addr:         startup.addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		comms.logs <- Log{Scope: "http", Msg: fmt.Sprintf("Listening locally -> %s", startup.addr)}

		if err := local.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			comms.logs <- Log{Scope: "http", Msg: "Local listener stopped", Err: err}
		}
	}()

	go func() {
		if err := http.Serve(listener, mux); err != nil && err != http.ErrServerClosed {
			comms.logs <- Log{Scope: "http", Msg: "Tunnel listener stopped", Err: err}
			stop()
		}
	}()

	<-signals.Done()
	stop()

	comms.logs <- Log{Scope: "http", Level: LevelWarn, Msg: "Shutdown signal received"}

	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := local.Shutdown(shutdown); err != nil {
		comms.logs <- Log{Scope: "http", Msg: "Local listener shutdown failed", Err: err}
	}
	if err := listener.Close(); err != nil {
		comms.logs <- Log{Scope: "http", Msg: "Tunnel shutdown failed", Err: err}
	}
	persistSchedule(&cfg.comms)

	close(done)
	unsubscribeChannels(callback, credentials.bearer)

	if err := db.Close(); err != nil {
		comms.logs <- Log{Scope: "db", Msg: "Failed to close database", Err: err}
	}
	printLog(Log{Scope: "http", Msg: "Shutdown complete", Ts: time.Now()})
}

func persistSchedule(comms *Comms) {
	state := readServerState(comms.rd)

	if err := saveSchedule(state.Schedule); err != nil {
		printLog(Log{Scope: "cron", Msg: "Failed to save schedule", Err: err, Ts: time.Now()})
		return
	}
	for _, job := range buildJobs(state.Schedule) {
		if job.Stopped {
			continue
		}
		printLog(Log{Scope: "cron", Ts: time.Now(), Msg: fmt.Sprintf("Saved %s -> %ds remaining", job.Name, job.Remaining)})
	}
}

func initComms(comms *Comms, dbComms *DbComms) {
	comms.rd = make(chan ReadReq)
	comms.writeWisdom = make(chan WriteQuote)
	comms.writeTkn = make(chan WriteToken)
	comms.refreshTkn = make(chan WriteToken)
	comms.writeSeen = make(chan string)
	comms.logs = make(chan Log)
	comms.points = make(chan UpdateQuotaPoints)
	comms.schedule = make(chan ScheduleTick)

	rdComms := DbReadComms{}
	rdComms.findTag = make(chan FindTag)
	rdComms.get = make(chan GetChannel)
	rdComms.getAll = make(chan GetAll)
	rdComms.unused = make(chan GetUnused)
	rdComms.popularComments = make(chan PopularComments)
	rdComms.popularReplies = make(chan PopularReplies)
	rdComms.replies = make(chan GetReplies)
	rdComms.comments = make(chan GetComments)
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

	initTaskComms(&dbComms.tasks)
}
