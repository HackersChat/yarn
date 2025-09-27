-include environ.inc
.PHONY: help deps dev build install image release test clean tr tr-merge

export CGO_ENABLED=0

VERSION ?= $(shell git describe --abbrev=0 --tags 2>/dev/null)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BRANCH ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
BUILD  ?= $(shell git show -s --pretty=format:%cI 2>/dev/null)

GOCMD=go

DESTDIR=/usr/local/bin

ifeq ($(LOCAL), 1)
IMAGE := r.mills.io/prologic/yarnd
else
IMAGE := prologic/yarnd
endif

ifeq ($(BRANCH), main)
TAG := latest
else
TAG := dev
endif

all: help

help: ## Show this help message
	@echo "Yarn.social - a Self-Hosted, Twitter™-like Decentralised microBlogging platform"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m\033[0m\n"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

deps: ## Install any required dependencies
	@$(GOCMD) install github.com/tdewolff/minify/v2/cmd/minify@latest
	@$(GOCMD) install github.com/nicksnyder/go-i18n/v2/goi18n@latest
	@$(GOCMD) install github.com/astaxie/bat@latest

dev: DEBUG=1
dev: build ## Build debug version of yarnc (cli) and yarnd (server)
	@./yarnc -v
	@./yarnd -D -O -R $(ARGS)

cli: ## Build the yarnc command-line client
ifeq ($(DEBUG), 1)
	@echo "Building in debug mode..."
	@$(GOCMD) build -tags "netgo static_build" -installsuffix netgo \
		-ldflags "\
		-X $(shell go list).Version=$(VERSION) \
		-X $(shell go list).Commit=$(COMMIT) \
		-X $(shell go list).Build=$(BUILD)" \
		./cmd/yarnc/
else
	@$(GOCMD) build -tags "netgo static_build" -installsuffix netgo \
		-ldflags "-w \
		-X $(shell go list).Version=$(VERSION) \
		-X $(shell go list).Commit=$(COMMIT) \
		-X $(shell go list).Build=$(BUILD)" \
		./cmd/yarnc/
endif

server: generate ## Build the yarnd server
ifeq ($(DEBUG), 1)
	@echo "Building in debug mode..."
	@$(GOCMD) build $(FLAGS) -tags "netgo static_build" -installsuffix netgo \
		-ldflags "\
		-X $(shell go list).Version=$(VERSION) \
		-X $(shell go list).Commit=$(COMMIT) \
		-X $(shell go list).Build=$(BUILD)" \
		./cmd/yarnd/...
else
	@$(GOCMD) build $(FLAGS) -tags "netgo static_build" -installsuffix netgo \
		-ldflags "-w \
		-X $(shell go list).Version=$(VERSION) \
		-X $(shell go list).Commit=$(COMMIT) \
		-X $(shell go list).Build=$(BUILD)" \
		./cmd/yarnd/...
endif

build: cli server ## Build the cli and the server

generate: ## Generate any code required by the build
	@if [ x"$(DEBUG)" = x"1"  ]; then		\
	  echo 'Running in debug mode...';	\
	else								\
	  minify -b -o ./internal/theme/static/css/yarn.min.css ./internal/theme/static/css/[0-9]*-*.css;	\
	  minify -b -o ./internal/theme/static/css/noscript.min.css ./internal/theme/static/css/noscript.css;	\
	  minify -b -o ./internal/theme/static/js/yarn.min.js ./internal/theme/static/js/[0-9]*-*.js;		\
	fi

install: build ## Install yarnc (cli) and yarnd (server) to $DESTDIR
	@install -D -m 755 yarnd $(DESTDIR)/yarnd
	@install -D -m 755 yarnc $(DESTDIR)/yarnc

ifeq ($(PUBLISH), 1)
image: generate ## Build the Docker image
	@docker buildx build \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg BUILD="$(BUILD)" \
		--platform linux/amd64,linux/arm64 --push -t $(IMAGE):$(TAG) .
else
image: generate
	@docker build  \
		--build-arg VERSION="$(VERSION)" \
		--build-arg COMMIT="$(COMMIT)" \
		--build-arg BUILD="$(BUILD)" \
		-t $(IMAGE):$(TAG) .
endif

release: generate ## Release a new version to Gitea
	@./tools/release.sh

dump_cache: ## Build dump_cache utility
	@CGO_ENABLED=0 $(GOCMD) build $(FLAGS) \
		-tags "netgo static_build" \
		-installsuffix netgo \
		./cmd/dump_cache/...

fmt: ## Format sources files
	@$(GOCMD) fmt ./...

# Can be used with `make test only=…` to limit the execution of unit tests to
# just the given subset. This is useful for debugging a failing test case. By
# default, all tests are executed.
only := ""

test: ## Run test suite
	@CGO_ENABLED=1 $(GOCMD) test -v -cover -race -run $(only) ./...

coverage: ## Get test coverage report
	@CGO_ENABLED=1 $(GOCMD) test -v -cover -race -cover -coverprofile=coverage.out  ./...
	@$(GOCMD) tool cover -html=coverage.out

tr: ## Build translations (i18n)
	@goi18n merge -outdir ./internal/langs ./internal/langs/active.*.toml
	@goi18n merge -outdir ./internal/langs ./internal/langs/active.*.toml ./internal/langs/translate.*.toml

clean: ## Remove untracked files
	@git clean -f -d -x

clean-all:  ## Remove untracked and Git ignored files
	@git clean -f -d -X
