GO ?= go

BIN_DIR ?= bin
GOBIN ?= $(shell $(GO) env GOPATH)/bin

.PHONY: help all build install test clean

help: ## Show available Make targets.
	@awk 'BEGIN { FS = ":.*## "; printf "Available targets:\n" } /^[a-zA-Z0-9_.-]+:.*## / { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

all: build ## Build all commands.

build: ## Build all commands into BIN_DIR, default ./bin.
	@mkdir -p "$(BIN_DIR)"
	$(GO) build -o "$(BIN_DIR)/ask-human" .
	$(GO) build -o "$(BIN_DIR)/notify-human" ./cmd/notify-human
	$(GO) build -o "$(BIN_DIR)/ask-human-config" ./cmd/ask-human-config
	@printf 'Built %s/ask-human\n' "$(BIN_DIR)"
	@printf 'Built %s/notify-human\n' "$(BIN_DIR)"
	@printf 'Built %s/ask-human-config\n' "$(BIN_DIR)"

install: ## Install all commands into GOBIN, default GOPATH/bin.
	@mkdir -p "$(GOBIN)"
	$(GO) build -o "$(GOBIN)/ask-human" .
	$(GO) build -o "$(GOBIN)/notify-human" ./cmd/notify-human
	$(GO) build -o "$(GOBIN)/ask-human-config" ./cmd/ask-human-config
	@printf 'Installed %s/ask-human\n' "$(GOBIN)"
	@printf 'Installed %s/notify-human\n' "$(GOBIN)"
	@printf 'Installed %s/ask-human-config\n' "$(GOBIN)"

test: ## Run the Go test suite.
	$(GO) test ./...

clean: ## Remove BIN_DIR.
	rm -rf "$(BIN_DIR)"
