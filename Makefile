.PHONY: check build vet clean

check: build vet

build:
	go build -o bot .

vet:
	go vet ./...

clean:
	rm -f bot
