# Stage 1: build
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o app ./cmd/server

# Stage 2: runtime
FROM alpine:3.19

WORKDIR /app
COPY --from=builder /app/app .
COPY web ./web

ENV HTTP_PORT=8080
ENV LOG_LEVEL=info

EXPOSE 8080

CMD ["./app"]
