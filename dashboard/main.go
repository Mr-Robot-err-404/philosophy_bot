package main

import (
	"embed"
	"flag"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed templates/*.html static/*
var assets embed.FS

func dbPath() string {
	if path := os.Getenv("DB_PATH"); path != "" {
		return path
	}
	return "../app.db"
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	path := flag.String("db", dbPath(), "path to app.db")
	flag.Parse()

	if _, err := os.Stat(*path); err != nil {
		log.Fatalf("database not found at %q: %v", *path, err)
	}
	db, err := openDB(*path)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	funcs := template.FuncMap{"since": since, "truncate": truncate}
	tmpl, err := template.New("index.html").Funcs(funcs).ParseFS(assets, "templates/*.html")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.FS(assets)))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := db.Ping(); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		stats, err := loadStats(db)
		if err != nil {
			log.Printf("stats: %v", err)
			http.Error(w, "failed to load stats", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, stats); err != nil {
			log.Printf("render: %v", err)
		}
	})

	srv := &http.Server{
		Addr:         *addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	log.Printf("dashboard on http://localhost%s (db: %s)", *addr, *path)
	log.Fatal(srv.ListenAndServe())
}

func truncate(n int, s string) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
