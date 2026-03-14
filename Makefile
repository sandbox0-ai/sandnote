BIN_DIR := bin
BIN := $(BIN_DIR)/sandnote

.PHONY: build test smoke clean

build:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/sandnote

test:
	go test ./...

smoke: build
	./scripts/smoke.sh

clean:
	rm -rf $(BIN_DIR)
