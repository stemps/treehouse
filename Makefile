BINARY := treehouse
GOCACHE ?= $(CURDIR)/.cache/go-build
export GOCACHE

.PHONY: test build clean

test:
	go test ./...

build:
	go build -o $(BINARY) .

clean:
	rm -rf $(BINARY) $(BINARY).exe .cache
