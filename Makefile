.PHONY: build test lint clean

build:
	go build -o bin/fev ./cmd/fev

test:
	go test ./... -race -timeout 120s

lint:
	golangci-lint run

clean:
	rm -rf bin/
