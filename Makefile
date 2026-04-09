VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/ashutosh/moor/cmd.version=$(VERSION)"

.PHONY: build test lint clean install

build:
	go build $(LDFLAGS) -o moor .

test:
	go test -v -race -cover ./...

lint:
	go vet ./...

clean:
	rm -f moor

install: build
	cp moor /usr/local/bin/moor

fmt:
	go fmt ./...
