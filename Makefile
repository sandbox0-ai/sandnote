BIN_DIR := bin
BIN := $(BIN_DIR)/sandnote
CODEX_HOME ?= $(HOME)/.codex
SKILL_SRC := $(CURDIR)/skills/sandnote
SKILL_DEST := $(CODEX_HOME)/skills/sandnote

.PHONY: build install install-bin install-skill test smoke clean

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/sandnote

install: install-bin install-skill

install-bin:
	go install ./cmd/sandnote
	@GOBIN="$${GOBIN:-$$(go env GOBIN)}"; \
	if [ -z "$$GOBIN" ]; then \
		GOBIN="$$(go env GOPATH)/bin"; \
	fi; \
	printf 'Installed sandnote binary to %s/sandnote\n' "$$GOBIN"

install-skill:
	mkdir -p $(CODEX_HOME)/skills
	ln -sfn $(SKILL_SRC) $(SKILL_DEST)
	@printf 'Linked sandnote skill to %s\n' "$(SKILL_DEST)"

test:
	go test ./...

smoke: build
	./scripts/smoke.sh

clean:
	rm -rf $(BIN_DIR)
