.PHONY: build test-unit test-functional run

build:
	go build -o bin/server ./cmd/server

test-unit:
	go test ./test/unit/... -count=1 -timeout 30s -coverprofile=coverage.out

test-functional:
	go test ./test/functional/... -count=1 -timeout 120s -v

run:
	go run ./cmd/server
