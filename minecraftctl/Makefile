.PHONY: build test clean install

BINARY_NAME=minecraftctl
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/paul/minecraftctl/internal/version.Version=$(VERSION)"

build:
	go build $(LDFLAGS) -o $(BINARY_NAME) ./cmd/minecraftctl

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)

install: build
	install -m 755 $(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet

