package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type LogLevel int

const (
	LevelAuto LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

const (
	ansiReset   = "\x1b[0m"
	ansiDim     = "\x1b[2m"
	ansiRed     = "\x1b[31m"
	ansiGreen   = "\x1b[32m"
	ansiYellow  = "\x1b[33m"
	ansiMagenta = "\x1b[35m"
	ansiCyan    = "\x1b[36m"
)

var colourEnabled = detectColour()

func detectColour() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func paint(text string, colour string) string {
	if !colourEnabled {
		return text
	}
	return colour + text + ansiReset
}

func (entry Log) level() LogLevel {
	if entry.Level != LevelAuto {
		return entry.Level
	}
	if entry.Err != nil {
		return LevelError
	}
	return LevelInfo
}

func levelName(level LogLevel) string {
	switch level {
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "INFO"
	}
}

func levelTag(level LogLevel) string {
	name := fmt.Sprintf("%-5s", levelName(level))

	switch level {
	case LevelWarn:
		return paint(name, ansiYellow)
	case LevelError:
		return paint(name, ansiRed)
	default:
		return paint(name, ansiGreen)
	}
}

func (entry Log) text() string {
	if entry.Err == nil {
		return entry.Msg
	}
	if entry.Msg == "" {
		return entry.Err.Error()
	}
	return entry.Msg + paint(" -> "+entry.Err.Error(), ansiRed)
}

func printLog(entry Log) {
	body := entry.text()
	if body == "" {
		return
	}
	ts := entry.Ts
	if ts.IsZero() {
		ts = time.Now()
	}
	fields := []string{paint(ts.Format("15:04:05"), ansiDim), levelTag(entry.level())}

	if entry.Scope != "" {
		fields = append(fields, paint(fmt.Sprintf("%-8s", entry.Scope), ansiCyan))
	}
	fields = append(fields, body)

	fmt.Println(strings.Join(fields, " "))
}

func printBanner(lines []string) {
	rule := paint(strings.Repeat("─", 46), ansiDim)

	fmt.Println(rule)
	for _, line := range lines {
		fmt.Println(paint("  "+line, ansiMagenta))
	}
	fmt.Println(rule)
}
