FROM golang:1.25.0-alpine AS builder

# Нужен компилятор C для go-sqlite3
RUN apk add --no-cache build-base

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=1 GOOS=linux GOARCH=amd64

RUN go build -o app ./cmd/server

FROM alpine:3.19

WORKDIR /app

RUN apk add --no-cache ca-certificates

COPY --from=builder /app/app .
COPY web ./web

COPY data ./data

RUN mkdir -p /app/data

ENV HTTP_PORT=8080
ENV LOG_LEVEL=info

EXPOSE 8080
CMD ["./app"]
