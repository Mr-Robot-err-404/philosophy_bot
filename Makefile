.PHONY: check build bot dashboard vet clean

check: build vet

build: bot dashboard

bot:
	go build -o bot .

dashboard:
	go build -o dashboard/dashboard ./dashboard

vet:
	go vet ./...

clean:
	rm -f bot dashboard/dashboard
