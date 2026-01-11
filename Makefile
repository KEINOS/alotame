build: test lint
	go build -o bin/alotame main.go

test:
	go test -race ./...
	golangci-lint run --fix

lint:
	golangci-lint run

# Update dependencies
update:
	go get -u ./...
	go mod tidy
