BINARY := agentspec
PKG    := ./cmd/agentspec
MODULE := github.com/szaher/agentspec

.PHONY: all build test lint fmt vet validate clean pre-commit fmt-examples

all: lint test build

## Pre-commit: run everything CI checks, fix what's fixable

pre-commit: fmt fmt-examples vet lint test build validate
	@echo "All checks passed. Ready to commit."

## Build

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(PKG)

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

## Validate & format examples

validate: build
	@for f in examples/*/*.ias; do \
		echo "Validating $$f..."; \
		./$(BINARY) validate "$$f" || exit 1; \
	done

fmt-examples: build
	@for f in examples/*/*.ias; do \
		./$(BINARY) fmt "$$f"; \
	done

## Operator / controller-gen

CONTROLLER_GEN ?= $(shell go env GOPATH)/bin/controller-gen

.PHONY: generate manifests controller-gen

generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="" paths="./internal/api/..."

manifests: controller-gen
	$(CONTROLLER_GEN) crd rbac:roleName=agentspec-operator-role webhook paths="./internal/api/..." paths="./internal/operator/..." output:crd:artifacts:config=config/crd/bases output:rbac:artifacts:config=config/rbac

controller-gen:
	@test -f $(CONTROLLER_GEN) || go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

## Clean

clean:
	rm -f $(BINARY)
	rm -f .agentspec.state.json
