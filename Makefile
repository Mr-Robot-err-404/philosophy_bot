CMD ?= server

.PHONY: check build bot dashboard run run-dashboard vet clean

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

vet:
	go vet ./...

clean:
	rm -f bot dashboard/dashboard
