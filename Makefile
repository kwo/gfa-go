BINARY := gfa
CMD := ./cmd/gfa
DIST := dist
VERSION ?= dev
LDFLAGS := -s -w -X main.buildVersion=$(VERSION)

.PHONY: all build clean test lint

all: test build

build:
	mkdir -p $(DIST)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf $(DIST)
