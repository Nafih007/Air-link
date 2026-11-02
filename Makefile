.PHONY: lint test build start dev tidy

lint:
	gofmt -l .
	go vet ./...

test:
	go test ./...

tidy:
	go mod tidy

build:
	wails build

start:
	wails dev

dev: start
