BINARY := gfa
CMD := ./cmd/gfa
DIST := dist

.PHONY: all build clean test lint

all: test build

build:
	mkdir -p $(DIST)
	go build -o $(DIST)/$(BINARY) $(CMD)

test:
	go test ./...

lint:
	golangci-lint run

clean:
	rm -rf $(DIST)
