GO ?= go

.PHONY: check test build

check:
	$(GO) fmt ./...
	$(GO) vet ./...
	$(GO) test ./...

test:
	$(GO) test ./...

build:
	$(GO) build -trimpath -o beforeup .
