.PHONY: test lint
test:
	go test -race ./...

lint:
	golangci-lint run ./...

local:
	go run ./cmd/main.go

local-frontend:
	cd frontend && npm run dev

build:
	cd frontend && npm run build
	go build -o page-insight ./cmd/main.go