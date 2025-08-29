SHELL := /bin/sh

APP := tasks-api
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP)

IMAGE ?= tasks-api-go:dev
CONTAINER ?= tasks-api

GOFLAGS ?=
LDFLAGS ?= -s -w

.PHONY: run test build clean docker-build docker-run docker-stop lint

run:
	go run $(GOFLAGS) .

test:
	go test ./... -v

build:
	mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) .

clean:
	rm -rf $(BIN_DIR)

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-stop
	docker run --rm -p 8080:8080 -p 8081:8081 --name $(CONTAINER) $(IMAGE)

docker-stop:
	-@docker rm -f $(CONTAINER) >/dev/null 2>&1 || true

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping lint"
