.PHONY: fmt fmt-check vet lint test test-race cover build check migrate-up migrate-down docker-up docker-down

fmt:
	gofmt -w ./cmd ./internal

fmt-check:
	test -z "$$(gofmt -l ./cmd ./internal)"

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./...

test-race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

build:
	go build -trimpath ./...

check: fmt-check vet lint test-race build

migrate-up:
	go run ./cmd/migrate -direction up

migrate-down:
	go run ./cmd/migrate -direction down -steps 1

docker-up:
	docker compose up --build -d

docker-down:
	docker compose down
