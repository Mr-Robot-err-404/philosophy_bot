package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

const StartupGrace = 60 * time.Second

func schedulePath() string {
	if path := os.Getenv("SCHEDULE_PATH"); path != "" {
		return path
	}
	return "./schedule.json"
}

func newSchedule(start time.Time) map[string]time.Time {
	schedule := make(map[string]time.Time, len(jobIntervals))

	for name, every := range jobIntervals {
		schedule[name] = start.Add(every)
	}
	return schedule
}

func loadSchedule() (map[string]time.Time, error) {
	saved := map[string]time.Time{}

	data, err := os.ReadFile(schedulePath())
	if err != nil {
		return saved, err
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return saved, fmt.Errorf("malformed schedule file %q: %w", schedulePath(), err)
	}
	return saved, nil
}

func saveSchedule(schedule map[string]time.Time) error {
	path := schedulePath()

	data, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".schedule-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func resumeSchedule(now time.Time) (map[string]time.Time, []string) {
	saved, err := loadSchedule()
	if err != nil {
		return newSchedule(now), nil
	}
	schedule := make(map[string]time.Time, len(jobIntervals))
	notes := []string{}

	for name, every := range jobIntervals {
		next, ok := saved[name]

		switch {
		case !ok || next.IsZero():
			schedule[name] = now.Add(every)
			notes = append(notes, fmt.Sprintf("%s scheduled fresh", name))
		case next.After(now):
			schedule[name] = next
			notes = append(notes, fmt.Sprintf("%s resumes in %v", name, time.Until(next).Round(time.Second)))
		default:
			schedule[name] = now.Add(StartupGrace)
			notes = append(notes, fmt.Sprintf("%s overdue by %v", name, now.Sub(next).Round(time.Second)))
		}
	}
	return schedule, notes
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
