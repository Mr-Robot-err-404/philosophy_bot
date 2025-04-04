package main

import (
	"bot/philosophy/internal/database"
	"context"
	"database/sql"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	_ "modernc.org/sqlite"
)

var db *sql.DB
var queries *database.Queries
var ctx context.Context

func connect_db(db_path string) error {
	var err error
	db, err = sql.Open("sqlite", db_path)
	if err != nil {
		return err
	}
	err = db.Ping()
	if err != nil {
		return err
	}
	ctx = context.Background()
	queries = database.New(db)
	return nil
}

func sisyphus() {
	err := connect_db("./app.db")
	if err != nil {
		log.Fatal(err)
	}
	err = godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}
}

func init_tables() error {
	err := generateQuotesTable("./public/quotes.csv")
	if err != nil {
		return err
	}
	id := "not_all_who_wander_are_lost"
	err = generateQuotaTable(id)
	if err != nil {
		return err
	}
	id = "sisyphus_smiled"
	err = generateLoginTable(id)
	return nil
}

func generateQuotaTable(id string) error {
	_, err := queries.SetupQuota(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func generateLoginTable(id string) error {
	_, err := queries.CreateLoginDetails(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

func generateQuotesTable(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	csv := strings.Split(string(data), "\n")

	for i := 1; i < len(csv); i++ {
		line := strings.TrimSpace(csv[i])
		if len(line) == 0 {
			continue
		}
		idx := 1
		quote := ""
		author := ""

		for {
			char := line[idx]
			if char == '"' {
				start := idx + 3
				end := len(line) - 1
				author = line[start:end]
				break
			}
			quote += string(char)
			idx++
		}
		id := int64(uuid.New().ID())
		params := database.CreateQuoteParams{ID: id, Quote: quote, Author: author, Categories: "default"}

		_, err := queries.CreateQuote(ctx, params)
		if err != nil {
			return err
		}
	}
	return nil
}
