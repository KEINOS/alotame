build: test lint
	go build -o bin/alotame main.go

.PHONY: test
test: download-deps test-main test-tool lint

test-main:
	@echo "* MAIN test:"; go test -race ./...

test-tool:
	@echo "* TOOL test:"; go test -short ./test/e2e/dns-resolver-check/...

# E2E integration test
test-e2e: test
	@trap 'rc=$$?; docker compose down --remove-orphans; exit $$rc' EXIT INT TERM; \
	docker compose pull && \
	docker compose build && \
	docker compose up --remove-orphans --wait alotame blocky && \
	sleep 5 && \
	docker compose run --rm e2e

lint: lint-main lint-tool

lint-main:
	@echo "* MAIN lint: golangci-lint run --fix ./..."
	@golangci-lint run --fix ./... 2>&1 | grep "0\ issues." || exit 1

lint-tool:
	@echo "* TOOL lint: golangci-lint run --fix ./test/e2e/dns-resolver-check"
	@golangci-lint run --fix ./test/e2e/dns-resolver-check 2>&1 | grep "0\ issues." || exit 1

# Update dependencies
update:
	go get -u ./...
	go work sync
	go mod tidy

download-deps:
	@go mod download
	@go work sync
