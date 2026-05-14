FROM golang:1.26 AS builder

WORKDIR /app

#RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
#RUN go install golang.org/x/tools/cmd/goimports@latest

COPY go.mod go.sum ./

RUN go mod download

COPY . .

#RUN test -z "$(gofmt -l .)"
#RUN test -z "$(goimports -l .)"
#RUN go vet ./...
#RUN /go/bin/golangci-lint run
#RUN go test ./...

RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/api/main.go

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/app .

EXPOSE 8080

CMD ["./app"]