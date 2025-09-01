SHELL := /bin/sh

APP := tasks-api
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP)

IMAGE ?= tasks-api-go:dev
CONTAINER ?= tasks-api

APP_PORT ?= 8080
HEALTH_PORT ?= 8081

GOFLAGS ?=
LDFLAGS ?= -s -w

.PHONY: run test build clean docker-build docker-run docker-run-detached docker-stop lint

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
	docker run --rm -p $(APP_PORT):8080 -p $(HEALTH_PORT):8081 --name $(CONTAINER) $(IMAGE)

docker-run-detached: docker-stop
	docker run -d -p $(APP_PORT):8080 -p $(HEALTH_PORT):8081 --name $(CONTAINER) $(IMAGE)

docker-stop:
	- docker rm -f $(CONTAINER) >/dev/null 2>&1 || true

lint:
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping lint"
