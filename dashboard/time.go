package main

import (
	"strconv"
	"time"
)

var layouts = []string{
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

func parseTime(raw string) (time.Time, bool) {
	if raw == "" {
		return time.Time{}, false
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts, true
		}
	}
	return time.Time{}, false
}

func prettyTime(raw string) string {
	ts, ok := parseTime(raw)
	if !ok {
		return "never"
	}
	return ts.Format("02 Jan 2006 15:04")
}

func since(raw string) string {
	ts, ok := parseTime(raw)
	if !ok {
		return ""
	}
	d := time.Since(ts)

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return pluralise(int(d.Minutes()), "min")
	case d < 24*time.Hour:
		return pluralise(int(d.Hours()), "hour")
	default:
		return pluralise(int(d.Hours()/24), "day")
	}
}

func pluralise(n int, unit string) string {
	if n == 1 {
		return "1 " + unit + " ago"
	}
	return strconv.Itoa(n) + " " + unit + "s ago"
}
