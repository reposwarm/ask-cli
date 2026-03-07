VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-X main.Version=$(VERSION)"

.PHONY: build test vet clean install

build:
	go build $(LDFLAGS) -o ask ./cmd/ask

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f ask

install:
	go install $(LDFLAGS) ./cmd/ask

all: vet test build
