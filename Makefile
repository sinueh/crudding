GO ?= go
BIN_DIR ?= bin
OUT_DIR ?= generated

.PHONY: all format test vet build install run genall clean

all: test build

format:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	mkdir -p "$(BIN_DIR)"
	$(GO) build -o "$(BIN_DIR)/crudding" ./cmd/crudding
	$(GO) build -o "$(BIN_DIR)/genall" ./cmd/genall

install:
	$(GO) install ./cmd/crudding

run:
	$(GO) run ./cmd/crudding $(ARGS)

genall:
	$(GO) run ./cmd/genall "$(OUT_DIR)"

clean:
	rm -rf "$(BIN_DIR)"