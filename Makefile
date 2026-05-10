fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	golangci-lint run

test:
	go test ./...

build-race:
	go build -race ./...

build:
	go build -o ./bin/app ./cmd/api/main.go

check: fmt vet lint build-race test

docker-build:
	docker build -t myapp .

all: check build