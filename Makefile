.PHONY: install-lint build test lint

# The name of your application
APP_NAME = reverse-proxy

# The go compiler to use
GO = go

# The golangci-lint tool
GOLINT = golangci-lint

install-lint:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 

build:
	$(GO) build -o $(APP_NAME)

test:
	$(GO) test -v ./...

lint: install-lint
	$(GOLINT) run ./...

clean:
	rm -f $(APP_NAME)

all: lint test build