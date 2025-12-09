.PHONY: build test clean install build-linux-amd64 build-all dist

BINARY_NAME=minecraftctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/paul/minecraftctl/internal/version.Version=$(VERSION)"
DIST_DIR=dist

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/minecraftctl

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 ./cmd/minecraftctl
	@echo "Built $(BINARY_NAME)-linux-amd64"

build-all: clean
	@mkdir -p $(DIST_DIR)
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/minecraftctl
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/minecraftctl
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/minecraftctl
	@echo "Builds complete in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

dist: build-all

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	rm -rf $(DIST_DIR)

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

