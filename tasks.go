package main

import (
	"bot/philosophy/internal/database"
	"bot/philosophy/internal/helper"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	TaskPollInterval = 30 * time.Second
	TaskBatchSize    = 5
	TaskMaxAttempts  = 3
	TaskRetryDelay   = 5 * time.Minute
	TaskHistory      = 30 * 24 * time.Hour
)

type CreateTask struct {
	params database.CreateTaskParams
	resp   chan TaskResp
}

type TaskResp struct {
	task database.Task
	err  error
}

type DueTasks struct {
	limit int64
	resp  chan DueTasksResp
}

type DueTasksResp struct {
	tasks []database.Task
	err   error
}

type ClaimTask struct {
	id   string
	resp chan TaskResp
}

type CompleteTask struct {
	params database.CompleteTaskParams
	resp   chan TaskResp
}

type FailTask struct {
	params database.FailTaskParams
	resp   chan TaskResp
}

type RetryTask struct {
	params database.RetryTaskParams
	resp   chan TaskResp
}

type ReleaseTasks struct {
	before time.Time
	resp   chan DueTasksResp
}

type PurgeTasks struct {
	before time.Time
	resp   chan DueTasksResp
}

type DbTaskComms struct {
	create   chan CreateTask
	due      chan DueTasks
	claim    chan ClaimTask
	complete chan CompleteTask
	fail     chan FailTask
	retry    chan RetryTask
	release  chan ReleaseTasks
	purge    chan PurgeTasks
}

func initTaskComms(tasks *DbTaskComms) {
	tasks.create = make(chan CreateTask)
	tasks.due = make(chan DueTasks)
	tasks.claim = make(chan ClaimTask)
	tasks.complete = make(chan CompleteTask)
	tasks.fail = make(chan FailTask)
	tasks.retry = make(chan RetryTask)
	tasks.release = make(chan ReleaseTasks)
	tasks.purge = make(chan PurgeTasks)
}

func createTask(params database.CreateTaskParams, ch chan<- CreateTask) TaskResp {
	resp := make(chan TaskResp)
	ch <- CreateTask{params: params, resp: resp}
	return <-resp
}

func dueTasks(limit int64, ch chan<- DueTasks) DueTasksResp {
	resp := make(chan DueTasksResp)
	ch <- DueTasks{limit: limit, resp: resp}
	return <-resp
}

func claimTask(id string, ch chan<- ClaimTask) TaskResp {
	resp := make(chan TaskResp)
	ch <- ClaimTask{id: id, resp: resp}
	return <-resp
}

func completeTask(id string, commentId string, ch chan<- CompleteTask) TaskResp {
	resp := make(chan TaskResp)
	params := database.CompleteTaskParams{ID: id, CommentID: sql.NullString{String: commentId, Valid: true}}
	ch <- CompleteTask{params: params, resp: resp}
	return <-resp
}

func failTask(id string, reason string, ch chan<- FailTask) TaskResp {
	resp := make(chan TaskResp)
	params := database.FailTaskParams{ID: id, Error: sql.NullString{String: reason, Valid: true}}
	ch <- FailTask{params: params, resp: resp}
	return <-resp
}

func retryTask(id string, reason string, at time.Time, ch chan<- RetryTask) TaskResp {
	resp := make(chan TaskResp)
	params := database.RetryTaskParams{ID: id, Error: sql.NullString{String: reason, Valid: true}, ActiveAt: at}
	ch <- RetryTask{params: params, resp: resp}
	return <-resp
}

func releaseTasks(before time.Time, ch chan<- ReleaseTasks) DueTasksResp {
	resp := make(chan DueTasksResp)
	ch <- ReleaseTasks{before: before, resp: resp}
	return <-resp
}

func purgeTasks(before time.Time, ch chan<- PurgeTasks) DueTasksResp {
	resp := make(chan DueTasksResp)
	ch <- PurgeTasks{before: before, resp: resp}
	return <-resp
}

func findQuote(id int64, quotes []database.Cornucopium) (database.Cornucopium, error) {
	for _, quote := range quotes {
		if quote.ID == id {
			return quote, nil
		}
	}
	return database.Cornucopium{}, fmt.Errorf("quote %d not found in cache", id)
}

func makeCommentParams(id string, quoteId int64) database.CreateCommentParams {
	return database.CreateCommentParams{ID: id, QuoteID: quoteId}
}

func enqueueTask(payload HookPayload, quote database.Cornucopium, comms *Comms, dbComms *DbComms) {
	delay := time.Duration(helper.RndInt(MinWait, MaxWait)) * time.Second
	active := time.Now().Add(delay)

	params := database.CreateTaskParams{
		ID:        uuid.New().String(),
		VideoID:   payload.VideoId,
		ChannelID: payload.ChannelId,
		QuoteID:   quote.ID,
		QuotaCost: COMMENT_COST,
		ActiveAt:  active,
	}
	resp := createTask(params, dbComms.tasks.create)

	if resp.err != nil {
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to enqueue video %s", payload.VideoId), Err: resp.err}
		return
	}
	state := readServerState(comms.rd)
	comms.points <- UpdateQuotaPoints{value: state.QuotaPoints - COMMENT_COST}

	comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Enqueued %s -> active in %v | quota -%d", payload.VideoId, delay.Round(time.Second), COMMENT_COST)}
}
