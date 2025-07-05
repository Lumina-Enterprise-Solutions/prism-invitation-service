# Makefile for prism-invitation-service
.DEFAULT_GOAL := help
.PHONY: help build run test test-all lint tidy docker-build clean

help: ## ✨ Show this help message
	@awk 'BEGIN {FS = ":.*?## "}; /^[\.a-zA-Z0-9_-]+:.*?## / {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## 🏗️  Build the application binary
	@echo ">> Building binary..."
	# FIX: Buat direktori bin jika belum ada dan beri nama binary yang benar
	@mkdir -p ./bin
	@go build -o ./bin/prism-invitation-service .

run: build ## 🚀 Run the application locally
	# FIX: Jalankan binary yang benar
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
	# FIX: Gunakan nama image yang benar
	@docker build -t lumina-enterprise-solutions/prism-invitation-service:latest -f ./Dockerfile .

# CLEAN
clean: ## 🗑️  Cleanup built artifacts
	@rm -rf ./bin
