BINARY := treehouse
GOCACHE ?= $(CURDIR)/.cache/go-build
GOFILES := $(shell find . -name '*.go' -not -path './vendor/*')
export GOCACHE

.PHONY: format format-check lint test check build clean

format:
	gofmt -w $(GOFILES)

format-check:
	@test -z "$$(gofmt -l $(GOFILES))" || { \
		echo "Go files need formatting. Run: make format"; \
		gofmt -l $(GOFILES); \
		exit 1; \
	}

lint:
	go vet ./...

test:
	go test ./...

check: format-check lint test

build:
	go build -o $(BINARY) .

clean:
	rm -rf $(BINARY) $(BINARY).exe .cache
