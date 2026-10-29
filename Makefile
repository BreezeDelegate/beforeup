GO ?= go

.PHONY: check fmt test build

check:
	@unformatted="$$(gofmt -l *.go)"; if [ -n "$$unformatted" ]; then printf 'Go files need gofmt:\n%s\n' "$$unformatted"; exit 1; fi
	$(GO) vet ./...
	$(GO) test ./...

fmt:
	gofmt -w *.go

test:
	$(GO) test ./...

build:
	$(GO) build -trimpath -o beforeup .
