.PHONY: build test test-e2e lint update

build: test lint
	go build -o bin/alotame main.go

test: download-deps
	@echo "\n* TOOL test:"; go test -short ./test/e2e/dns-resolver-check/...
	@echo "\n* MAIN test:"; go test -race ./...
	@echo "\n* LINT test:"; golangci-lint run --fix 2>&1 | grep "0\ issues." || exit 1

# E2E integration test
test-e2e:
	@trap 'rc=$$?; docker compose down --remove-orphans; exit $$rc' EXIT INT TERM; \
	go test -v ./test/e2e/dns-resolver-check/... && \
	docker compose build --no-cache && \
	docker compose up --remove-orphans --wait alotame blocky && \
	docker compose run --rm e2e

lint:
	@echo "* ROOT lint: golangci-lint run --fix ./..."
	@golangci-lint run --fix ./... 2>&1 | grep "0\ issues." || exit 1
	@echo "* E2E lint: golangci-lint run --fix ./test/e2e/dns-resolver-check"
	@golangci-lint run --fix ./test/e2e/dns-resolver-check 2>&1 | grep "0\ issues." || exit 1

# Update dependencies
update:
	go get -u ./...
	go work sync
	go mod tidy

download-deps:
	go mod download
	go work sync
