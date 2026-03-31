BINARY  := dev-assist
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -s -w"
OUTDIR  := bin

IMAGE   ?= ghcr.io/dev-sre-toolset/dev-assist

.PHONY: all build build-all docker docker-push release clean install tidy help

all: build

## build: build for the current platform
build:
	@mkdir -p $(OUTDIR)
	go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY) .

## build-all: cross-compile for macOS (amd64/arm64) and Linux (amd64/arm64)
build-all:
	@mkdir -p $(OUTDIR)
	GOOS=darwin  GOARCH=amd64 go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY)-darwin-arm64 .
	GOOS=linux   GOARCH=amd64 go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64 go build $(LDFLAGS) -o $(OUTDIR)/$(BINARY)-linux-arm64 .
	@echo "Built binaries in $(OUTDIR)/"
	@ls -lh $(OUTDIR)/

## release TAG=vX.Y: tag, build all platforms, and publish a GitHub release
##   Example: make release TAG=v0.1
release: build-all
	@if [ -z "$(TAG)" ]; then echo "usage: make release TAG=v0.1" && exit 1; fi
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	gh release create $(TAG) \
		$(OUTDIR)/$(BINARY)-darwin-amd64 \
		$(OUTDIR)/$(BINARY)-darwin-arm64 \
		$(OUTDIR)/$(BINARY)-linux-amd64 \
		$(OUTDIR)/$(BINARY)-linux-arm64 \
		--title "$(BINARY) $(TAG)" \
		--generate-notes

## install: install binary to GOPATH/bin
install:
	go install $(LDFLAGS) .

## tidy: tidy and download dependencies
tidy:
	go mod tidy

## docker: build the Docker image locally
docker:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

## docker-push: build and push the Docker image
docker-push: docker
	docker push $(IMAGE):$(VERSION)

## clean: remove built binaries
clean:
	rm -rf $(OUTDIR)

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //'
