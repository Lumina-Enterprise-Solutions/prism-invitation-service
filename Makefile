# Makefile for prism-file-service
.DEFAULT_GOAL := help
.PHONY: help build run test test-integration test-all lint tidy docker-build clean

help: ## ✨ Show this help message
	@awk 'BEGIN {FS = ":.*?## "}; /^[\.a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## 🏗️  Build the application binary
	@echo ">> Building binary..."
	@go build -o ./bin/prism-file-service .

run: build ## 🚀 Run the application locally
	@./bin/prism-invitation-service

tidy: ## 🧹 Tidy go module dependencies
	@go mod tidy -v

# TESTING
test: ## 🧪 Run unit tests only
	@echo ">> Running unit tests..."
	@go test -v -race -cover ./...


lint: ## 🧹 Run golangci-lint
	@golangci-lint run ./...

# DOCKER
docker-build: ## 🐳 Build the Docker image for this service
	@docker build -t lumina-enterprise-solutions/prism-file-service:latest -f ./Dockerfile .

# CLEAN
clean: ## 🗑️  Cleanup built artifacts
	@rm -rf ./bin
