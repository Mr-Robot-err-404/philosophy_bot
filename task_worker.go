package main

import (
	"context"
	"fmt"
	"time"
)

func taskWorker(ctx context.Context, comms *Comms, dbComms *DbComms) {
	ticker := time.NewTicker(TaskPollInterval)
	defer ticker.Stop()

	recoverTasks(comms, dbComms)

	for {
		select {
		case <-ctx.Done():
			comms.logs <- Log{Scope: "task", Msg: "Worker stopped"}
			return

		case <-ticker.C:
			sweepLeases(comms, dbComms)
			drainTasks(ctx, comms, dbComms)
		}
	}
}

func recoverTasks(comms *Comms, dbComms *DbComms) {
	orphans := runningTasks(dbComms.tasks.running)

	if orphans.err != nil {
		comms.logs <- Log{Scope: "task", Msg: "Failed to read running tasks", Err: orphans.err}
	}
	for _, task := range orphans.tasks {
		held := "unknown"
		if task.ClaimedAt.Valid {
			held = time.Since(task.ClaimedAt.Time).Round(time.Second).String()
		}
		comms.logs <- Log{Scope: "task", Level: LevelWarn, Msg: fmt.Sprintf("Orphan %s -> video %s held %s on attempt %d", task.ID, task.VideoID, held, task.Attempts)}
	}
	resp := releaseTasks(time.Now().UTC(), dbComms.tasks.release)

	if resp.err != nil {
		comms.logs <- Log{Scope: "task", Msg: "Failed to release stale tasks", Err: resp.err}
		return
	}
	if len(resp.tasks) > 0 {
		comms.logs <- Log{Scope: "task", Level: LevelWarn, Msg: fmt.Sprintf("Recovered %d interrupted tasks", len(resp.tasks))}
	}
	purged := purgeTasks(time.Now().Add(-TaskHistory), dbComms.tasks.purge)

	if purged.err != nil {
		comms.logs <- Log{Scope: "task", Msg: "Failed to purge old tasks", Err: purged.err}
		return
	}
	if len(purged.tasks) > 0 {
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Purged %d tasks older than %v", len(purged.tasks), TaskHistory)}
	}
}

func sweepLeases(comms *Comms, dbComms *DbComms) {
	resp := releaseTasks(time.Now().Add(-TaskLease).UTC(), dbComms.tasks.release)

	if resp.err != nil {
		comms.logs <- Log{Scope: "task", Msg: "Failed to sweep expired leases", Err: resp.err}
		return
	}
	for _, task := range resp.tasks {
		comms.logs <- Log{Scope: "task", Level: LevelWarn, Msg: fmt.Sprintf("Lease expired on %s -> video %s requeued after %v", task.ID, task.VideoID, TaskLease)}
	}
}

func drainTasks(ctx context.Context, comms *Comms, dbComms *DbComms) {
	resp := dueTasks(TaskBatchSize, dbComms.tasks.due)

	if resp.err != nil {
		comms.logs <- Log{Scope: "task", Msg: "Failed to read due tasks", Err: resp.err}
		return
	}
	if len(resp.tasks) == 0 {
		return
	}
	comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Working %d due tasks", len(resp.tasks))}

	for _, due := range resp.tasks {
		select {
		case <-ctx.Done():
			return
		default:
		}
		runTask(comms, dbComms, due.ID)
	}
}

func runTask(comms *Comms, dbComms *DbComms, id string) {
	claimed := claimTask(id, dbComms.tasks.claim)

	if claimed.err != nil {
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to claim %s", id), Err: claimed.err}
		return
	}
	task := claimed.task
	state := readServerState(comms.rd)

	quote, err := findQuote(task.QuoteID, state.Quotes)
	if err != nil {
		settleFailure(comms, dbComms, task.ID, err.Error())
		return
	}
	payload := CommentPayload{}
	payload.Snippet.ChannelId = task.ChannelID
	payload.Snippet.VideoId = task.VideoID
	payload.Snippet.TopLevelComment.Snippet.TextOriginal = constructWisdom(quote.Quote, quote.Author)

	info := CommentInfo{VideoId: task.VideoID, ChannelId: task.ChannelID, QuoteId: task.QuoteID, Payload: payload}
	comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Posting %s -> video %s", quote.Author, task.VideoID)}

	results := make(chan TaskResult, 1)
	postComment(info, state.Credentials, results)
	result := <-results

	if result.Err != nil {
		if task.Attempts >= TaskMaxAttempts {
			settleFailure(comms, dbComms, task.ID, result.Err.Error())
			return
		}
		next := time.Now().Add(TaskRetryDelay)
		again := retryTask(task.ID, result.Err.Error(), next, dbComms.tasks.retry)

		if again.err != nil {
			comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to reschedule %s", task.ID), Err: again.err}
			return
		}
		comms.spend <- Spend{cost: COMMENT_COST, reason: "retry"}

		comms.logs <- Log{Scope: "task", Level: LevelWarn, Msg: fmt.Sprintf("Attempt %d/%d failed, retrying in %v | quota -%d -> %v", task.Attempts, TaskMaxAttempts, TaskRetryDelay, COMMENT_COST, result.Err)}
		return
	}
	done := completeTask(task.ID, result.Id, dbComms.tasks.complete)

	if done.err != nil {
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to complete %s", task.ID), Err: done.err}
		return
	}
	comms.writeSeen <- task.VideoID

	if err := simpleMan(task.VideoID, dbComms.saveVid); err != nil {
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to record video %s", task.VideoID), Err: err}
	}
	dbComms.saveUsage <- Usage{channelId: task.ChannelID, quoteId: task.QuoteID}
	dbComms.saveComment <- makeCommentParams(result.Id, task.QuoteID)

	comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Posted comment %s -> video %s", result.Id, task.VideoID)}
}

func settleFailure(comms *Comms, dbComms *DbComms, id string, reason string) {
	resp := failTask(id, reason, dbComms.tasks.fail)

	if resp.err != nil {
		comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Failed to mark %s failed", id), Err: resp.err}
		return
	}
	comms.logs <- Log{Scope: "task", Msg: fmt.Sprintf("Task %s gave up -> %s", id, reason)}
}
