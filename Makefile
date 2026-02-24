BINARY := agentspec
PKG    := ./cmd/agentspec
MODULE := github.com/szaher/designs/agentz

.PHONY: all build test lint fmt vet validate clean

all: lint test build

## Build

build:
	go build -o $(BINARY) $(PKG)

## Test

test:
	go test ./... -count=1

test-v:
	go test ./... -count=1 -v

test-race:
	go test ./... -count=1 -race

## Lint

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:" && gofmt -l . && exit 1)

vet:
	go vet ./...

## Validate examples

validate: build
	@for f in examples/*/*.ias; do \
		echo "Validating $$f..."; \
		./$(BINARY) validate "$$f" || exit 1; \
	done

## Clean

clean:
	rm -f $(BINARY)
	rm -f .agentspec.state.json
