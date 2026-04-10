VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | tr -cd '[:alnum:]._-' || echo "dev")
LDFLAGS := -ldflags "-w -s -X github.com/MrHalder/moor/cmd.version=$(VERSION)"

.PHONY: build test lint clean install fmt

build:
	CGO_ENABLED=0 go build -trimpath $(LDFLAGS) -o moor .

test:
	go test -v -race -cover ./...

lint:
	go vet ./...

clean:
	rm -f moor

install: build
	install -m 755 moor /usr/local/bin/moor

fmt:
	go fmt ./...
