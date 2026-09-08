package main

import (
	"sort"
	"time"
)

const (
	TrendingInterval = 30 * time.Minute
	RefreshInterval  = 50 * time.Minute
	StatsInterval    = 21 * time.Hour
	QuotaInterval    = 25 * time.Hour
)

type ScheduleTick struct {
	job   string
	every time.Duration
}

type Job struct {
	Name      string `json:"name"`
	Interval  int    `json:"interval_seconds"`
	NextRun   string `json:"next_run"`
	Remaining int    `json:"seconds_remaining"`
	Stopped   bool   `json:"stopped"`
}

var jobIntervals = map[string]time.Duration{
	"trending": TrendingInterval,
	"refresh":  RefreshInterval,
	"stats":    StatsInterval,
	"quota":    QuotaInterval,
}

func newSchedule(start time.Time) map[string]time.Time {
	schedule := make(map[string]time.Time, len(jobIntervals))

	for name, every := range jobIntervals {
		schedule[name] = start.Add(every)
	}
	return schedule
}

func buildJobs(schedule map[string]time.Time) []Job {
	jobs := make([]Job, 0, len(jobIntervals))

	for name, every := range jobIntervals {
		job := Job{Name: name, Interval: int(every.Seconds())}
		next, ok := schedule[name]

		if !ok || next.IsZero() {
			job.Stopped = true
			jobs = append(jobs, job)
			continue
		}
		remaining := int(time.Until(next).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		job.NextRun = next.UTC().Format(time.RFC3339)
		job.Remaining = remaining
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].Remaining < jobs[j].Remaining })

	return jobs
}
