# rr-dns Makefile - minimal build/test tooling

BINARY_NAME=rr-dnsd
CMD_DIR=./cmd/rr-dnsd

.PHONY: all build test bench fmt vet lint clean ci

all: clean fmt vet test lint sec build

build:
	go build -o $(BINARY_NAME) $(CMD_DIR)

test:
	go test -v ./...

bench:
	go test -bench=. -benchmem ./...

fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run

sec:
	gosec ./...

clean:
	go clean
	rm -f $(BINARY_NAME)

cover:
	go test -coverprofile=cover.out ./...
	go tool cover -html=cover.out
