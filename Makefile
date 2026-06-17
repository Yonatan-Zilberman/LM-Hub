.PHONY: build run test lint clean

BINARY_NAME=lmhub

build:
	go build -o $(BINARY_NAME) ./cmd/lmhub

run: build
	./$(BINARY_NAME)

test:
	go test ./... -v

lint:
	go vet ./...

clean:
	rm -f $(BINARY_NAME)
	go clean
