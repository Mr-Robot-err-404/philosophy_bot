CMD ?= server
DB ?= app.db
MIGRATIONS ?= sql/schema
GOOSE ?= $(shell command -v goose 2>/dev/null || echo "go run github.com/pressly/goose/v3/cmd/goose@v3.24.2")
SQLC ?= $(shell command -v sqlc 2>/dev/null || echo "go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.28.0")

.PHONY: check build bot dashboard run run-dashboard generate up down status vet clean

check: build vet

build: bot dashboard

bot:
	go build -o bot .

dashboard:
	go build -o dashboard/dashboard ./dashboard

run: bot
	./bot $(CMD)

run-dashboard: dashboard
	./dashboard/dashboard -db app.db

generate:
	$(SQLC) generate

up:
	$(GOOSE) -dir $(MIGRATIONS) sqlite3 $(DB) up

down:
	$(GOOSE) -dir $(MIGRATIONS) sqlite3 $(DB) down

status:
	$(GOOSE) -dir $(MIGRATIONS) sqlite3 $(DB) status

vet:
	go vet ./...

clean:
	rm -f bot dashboard/dashboard
