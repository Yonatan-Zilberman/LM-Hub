.PHONY: build build-treesitter run test test-treesitter lint clean install release

BINARY_NAME=lmh

build:
	go build -o $(BINARY_NAME) ./cmd/lmh

build-treesitter:
	go build -tags treesitter -o $(BINARY_NAME) ./cmd/lmh

run: build
	./$(BINARY_NAME)

test:
	go test ./... -v

test-unit:
	go test ./internal/config/... ./internal/modelmanager/... -count=1 -v

test-api:
	go test ./internal/api/... -count=1 -v

test-modes:
	go test ./internal/modes/... -count=1 -v

test-cli:
	go test ./cmd/lmh/... -count=1 -v

test-all: test-unit test-api test-modes test-cli

qa-smoke:
	./scripts/qa-manual.sh

test-treesitter:
	go test -tags treesitter ./... -v

lint:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)
	go clean

install:
	go install ./cmd/lmh
	ln -sf $(shell go env GOPATH)/bin/lmh $(shell go env GOPATH)/bin/lmhub

release:
	goreleaser release --snapshot --clean
