GO ?= go
GOFMT ?= gofmt

.PHONY: check fmt test build

check:
	@command -v $(GOFMT) >/dev/null 2>&1 || { echo "gofmt is required for make check"; exit 1; }
	@unformatted="$$( $(GOFMT) -l *.go )" || exit 1; if [ -n "$$unformatted" ]; then printf 'Go files need gofmt:\n%s\n' "$$unformatted"; exit 1; fi
	$(GO) vet ./...
	$(GO) test ./...

fmt:
	$(GOFMT) -w *.go

test:
	$(GO) test ./...

build:
	$(GO) build -trimpath -o beforeup .
