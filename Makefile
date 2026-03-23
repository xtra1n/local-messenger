.PHONY: build run test test-coverage lint fmt clean help deps

BINARY_NAME=local-messenger
CMD_PATH=./cmd/server
MAIN_GO=$(CMD_PATH)/main.go

help:
	@echo "Available commands:"
	@echo "  make build      - Build the binary"
	@echo "  make run        - Run the server"
	@echo "  make test       - Run tests"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make lint       - Run linter"
	@echo "  make fmt        - Format code"
	@echo "  make clean      - Remove binary"
	@echo "  make deps       - Install dependencies"

deps:
	go mod download
	@echo "Installing golangci-lint..."
	which golangci-lint || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

build:
	go build -v -o $(BINARY_NAME) $(MAIN_GO)

run: build
	./$(BINARY_NAME)

test:
	go test -v ./...

test-coverage:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

lint:
	golangci-lint run ./...

fmt:
	go fmt ./...
	goimports -local github.com/xtra1n/local-messenger -w .

clean:
	rm -f $(BINARY_NAME) coverage.out coverage.html
	go clean -cache
