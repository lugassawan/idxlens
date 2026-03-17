.PHONY: build custom-gcl lint fmt test coverage clean init

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
MODULE  := github.com/lugassawan/idxlens
LDFLAGS := -s -w \
	-X '$(MODULE)/internal/cli.Version=$(VERSION)' \
	-X '$(MODULE)/internal/cli.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/cli.Date=$(DATE)'

build:
	go build -ldflags="$(LDFLAGS)" -o bin/idxlens ./cmd/idxlens

custom-gcl:
	golangci-lint custom

lint: custom-gcl
	./custom-gcl run ./...

fmt:
	gofmt -w .
	golines -w --max-len=120 .

test:
	go test ./...

coverage:
	mkdir -p coverage
	go test -race -coverprofile=coverage/coverage.out ./...
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html

clean:
	rm -rf bin/ dist/ coverage/ custom-gcl

init:
	mise trust
	mise install
	mise reshim
	$(MAKE) custom-gcl
	git config core.hooksPath .githooks
