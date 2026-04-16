.PHONY: test lint
test:
	go test -race ./...

lint:
	golangci-lint run ./...

local:
	go run ./cmd/main.go

build:
	go build -o page-insight ./cmd/main.go