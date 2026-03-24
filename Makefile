SHELL := /bin/bash

APP_NAME := accounting
BINARY := $(APP_NAME)
IMAGE ?= $(APP_NAME)
TAG ?= latest
APP_PORT ?= 8080
HOST_PORT ?= 8080
CONTAINER_PORT ?= 80
DATA_DIR ?= $(PWD)/data
DB_PATH ?= $(DATA_DIR)/accounting.db

.PHONY: help build test run clean docker-build docker-run docker-push

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "%-20s %s\n", $$1, $$2}'

build: ## Build the accounting binary
	go build -o $(BINARY) .

test: ## Run all Go tests
	go test ./...

run: ## Run the service locally with Go
	mkdir -p $(DATA_DIR)
	DB_PATH=$(DB_PATH) PORT=$(APP_PORT) go run .

clean: ## Remove built artifacts
	rm -f $(BINARY)

docker-build: ## Build the Docker image
	docker build -t $(IMAGE):$(TAG) .

docker-run: docker-build ## Run the Docker image locally
	mkdir -p $(DATA_DIR)
	@test -n "$$AUTH_USER" || (echo "AUTH_USER environment variable is required" >&2 && exit 1)
	@test -n "$$AUTH_PASS" || (echo "AUTH_PASS environment variable is required" >&2 && exit 1)
	docker run --rm -p $(HOST_PORT):$(CONTAINER_PORT) -v $(DATA_DIR):/data \
		-e DB_PATH=/data/accounting.db \
		-e PORT=$(CONTAINER_PORT) \
		-e AUTH_USER=$$AUTH_USER \
		-e AUTH_PASS=$$AUTH_PASS \
		$(IMAGE):$(TAG)

docker-push: ## Push the Docker image to the configured registry
	docker push $(IMAGE):$(TAG)
