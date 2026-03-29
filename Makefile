.PHONY: build test fmt vet lint lint-fix run-help test-cover sync-benchmark-offline-snapshot

GOLANGCI_LINT ?= golangci-lint

build:
	go build -o bin/lazy-tool ./cmd/lazy-tool

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out -o coverage.html

fmt:
	go fmt ./...

vet:
	go vet ./...

# Requires golangci-lint v2+ on PATH:
#   go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
lint: vet
	$(GOLANGCI_LINT) run ./...

lint-fix: vet
	$(GOLANGCI_LINT) run --fix ./...

run-help:
	go run ./cmd/lazy-tool --help

# After editing benchmark/golden/sample_benchmark_rows.jsonl, refresh the JSON array mirror for PR-friendly diffs:
sync-benchmark-offline-snapshot:
	python3 benchmark/scripts/sync_offline_benchmark_snapshot.py
