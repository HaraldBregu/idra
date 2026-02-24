VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
BINARY  := idra
GOFLAGS := CGO_ENABLED=0

.PHONY: build dev clean cross test proto agents-deps

## build: compile for the current platform
build:
	$(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/idra

## dev: build and run in foreground
dev: build
	./$(BINARY) run

## test: run all tests
test:
	go test ./...

## clean: remove build artifacts
clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf dist/

## cross: build for all supported platforms
cross:
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 $(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64       ./cmd/idra
	GOOS=linux   GOARCH=arm64 $(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64       ./cmd/idra
	GOOS=darwin  GOARCH=amd64 $(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64      ./cmd/idra
	GOOS=darwin  GOARCH=arm64 $(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64      ./cmd/idra
	GOOS=windows GOARCH=amd64 $(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe ./cmd/idra
	GOOS=windows GOARCH=arm64 $(GOFLAGS) go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-arm64.exe ./cmd/idra
	@echo "Built binaries in dist/"
	@ls -lh dist/

## proto: regenerate protobuf stubs (requires protoc + plugins)
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
	       proto/agent.proto

## agents-deps: install Python and Node agent dependencies
agents-deps:
	cd agents/python-summarizer && pip install -r requirements.txt
	cd agents/ts-sentiment && npm install

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
