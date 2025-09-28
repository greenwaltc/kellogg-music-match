# Makefile for Kellogg Music Match Backend
# This Makefile provides targets for OpenAPI code generation, testing, and common development tasks

.PHONY: help generate generate-sqlc clean build test test-ginkgo test-behavioral test-quick run docker-build docker-run docker-stop dev install-tools

# Default target
help: ## Show this help message
	@echo "Kellogg Music Match Backend - Available targets:"
	@echo "Features: OpenAPI generation, Ticketmaster API, PostgreSQL, Ginkgo testing"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Testing commands:"
	@echo "  test             Run all tests (43 Ginkgo/Gomega BDD tests)"
	@echo "  test-quick       Run quick Go unit tests only"
	@echo "  test-ginkgo      Run Ginkgo behavioral tests"
	@echo "  test-behavioral  Alias for test-ginkgo"
	@echo ""

generate: generate-sqlc ## Generate OpenAPI server code and SQLC code from specifications
	@echo "🔧 Generating OpenAPI server code..."
	@# Remove existing generated files temporarily
	@if [ -d "generated/temp_backup" ]; then rm -rf generated/temp_backup; fi
	@mkdir -p generated/temp_backup
	@if [ -d "generated" ] && [ "$$(ls -A generated 2>/dev/null)" ]; then \
		cp generated/*.go generated/temp_backup/ 2>/dev/null || true; \
	fi
	@# Generate fresh OpenAPI code
	@mkdir -p generated/temp_gen
	@docker run --rm -v $$(pwd):/local openapitools/openapi-generator-cli generate \
		-i /local/openapi.yaml \
		-g go-server \
		-o /local/generated/temp_gen \
		--additional-properties=packageName=generated
	@# Move generated Go files to the generated package (excluding main.go)
	@cp generated/temp_gen/go/*.go generated/
	@rm -f generated/main.go  # We have our own main.go in cmd/
	@# Clean up temporary directories
	@rm -rf generated/temp_gen generated/temp_backup
	@echo "✅ OpenAPI code generation complete!"
	@echo "📦 Generated files are in the 'generated' package"
	@echo "💼 Business logic remains in the 'business' package"
	@echo "🚀 Application entry point is in 'cmd/main.go'"

generate-sqlc: ## Generate SQLC code from SQL schema and queries
	@echo "🗄️ Generating SQLC code..."
	@if command -v sqlc >/dev/null 2>&1; then \
		sqlc generate; \
		echo "✅ SQLC code generation complete!"; \
	elif [ -f ~/go/bin/sqlc ]; then \
		~/go/bin/sqlc generate; \
		echo "✅ SQLC code generation complete!"; \
	else \
		echo "❌ 'sqlc' tool not found. Installing..."; \
		go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest; \
		~/go/bin/sqlc generate; \
		echo "✅ SQLC installed and code generated!"; \
	fi

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning build artifacts..."
	@rm -f server backend
	@rm -rf db/sqlc/*.go
	@echo "✅ Cleanup complete!"
	@rm -rf generated/temp_gen generated/temp_backup
	@rm -f *.log
	@go clean
	@echo "✅ Clean complete!"

build: generate-sqlc ## Build the backend server (includes SQLC generation)
	@echo "🔨 Building backend server..."
	@go mod tidy
	@go build -o server ./cmd
	@echo "✅ Build complete! Binary: ./server"

test: ## Run all tests (Go tests + Ginkgo behavioral tests)
	@echo "🧪 Running Go tests..."
	@go test -p 1 ./... -v
	@echo "🎭 Running Ginkgo behavioral tests..."
	@$(MAKE) test-ginkgo
	@echo "✅ All tests complete!"

test-quick: ## Run quick Go unit tests only
	@echo "🧪 Running quick Go tests..."
	@go test -p 1 ./... -short
	@echo "✅ Quick tests complete!"

test-ginkgo: ## Run Ginkgo behavioral tests
	@echo "🎭 Running Ginkgo behavioral tests..."
	@if command -v ginkgo >/dev/null 2>&1; then \
		ginkgo run ./...; \
	elif [ -f ~/go/bin/ginkgo ]; then \
		~/go/bin/ginkgo run ./...; \
	else \
		echo "❌ 'ginkgo' tool not found. Installing..."; \
		go install github.com/onsi/ginkgo/v2/ginkgo@latest; \
		~/go/bin/ginkgo run ./...; \
	fi
	@echo "✅ Behavioral tests complete!"

test-behavioral: test-ginkgo ## Alias for test-ginkgo

run: build ## Build and run the backend server (includes SQLC generation)
	@echo "🌍 Setting database environment variables..."
	@export KELLOGG_MUSIC_MATCH_DATABASE_HOST=localhost && \
	export KELLOGG_MUSIC_MATCH_DATABASE_PORT=5432 && \
	export KELLOGG_MUSIC_MATCH_DATABASE_NAME=kellogg_music_match && \
	export KELLOGG_MUSIC_MATCH_DATABASE_USER=postgres && \
	export KELLOGG_MUSIC_MATCH_DATABASE_PASSWORD=postgres123 && \
	export KELLOGG_MUSIC_MATCH_DATABASE_SSL_MODE=disable && \
	echo "🚀 Starting server on :8080..." && \
	./server

dev: generate-sqlc ## Start development server with live reload (includes SQLC generation)
	@echo "🌍 Setting database environment variables..."
	@export KELLOGG_MUSIC_MATCH_DATABASE_HOST=localhost && \
	export KELLOGG_MUSIC_MATCH_DATABASE_PORT=5432 && \
	export KELLOGG_MUSIC_MATCH_DATABASE_NAME=kellogg_music_match && \
	export KELLOGG_MUSIC_MATCH_DATABASE_USER=postgres && \
	export KELLOGG_MUSIC_MATCH_DATABASE_PASSWORD=postgres123 && \
	export KELLOGG_MUSIC_MATCH_DATABASE_SSL_MODE=disable && \
	if command -v air >/dev/null 2>&1; then \
		echo "🔥 Starting development server with live reload..."; \
		air; \
	else \
		echo "❌ 'air' tool not found. Install with: make install-tools"; \
		echo "🚀 Starting server normally..."; \
		$(MAKE) run; \
	fi

docker-build: ## Build Docker image for backend
	@echo "🐳 Building Docker image..."
	@cd .. && docker-compose build backend
	@echo "✅ Docker image built!"

docker-run: ## Start the full application with Docker Compose
	@echo "🐳 Starting application with Docker Compose..."
	@cd .. && docker-compose up -d
	@echo "✅ Application started!"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend API: http://localhost:8080"
	@echo "📊 Health Check: http://localhost:8080/health"

docker-stop: ## Stop Docker Compose services
	@echo "🛑 Stopping Docker Compose services..."
	@cd .. && docker-compose down
	@echo "✅ Services stopped!"

docker-logs: ## Show Docker Compose logs
	@echo "📋 Showing application logs..."
	@cd .. && docker-compose logs -f

install-tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	@go install github.com/air-verse/air@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	@go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@go install github.com/onsi/gomega@latest
	@echo "✅ Development tools installed!"

lint: ## Run linting
	@echo "🔍 Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "❌ golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
		echo "🔍 Running basic go vet..."; \
		go vet ./...; \
	fi
	@echo "✅ Linting complete!"

format: ## Format Go code
	@echo "📝 Formatting code..."
	@go fmt ./...
	@echo "✅ Code formatting complete!"

deps: ## Download and tidy dependencies
	@echo "📦 Managing dependencies..."
	@go mod download
	@go mod tidy
	@echo "✅ Dependencies updated!"

check: clean lint test ## Run full check (clean, lint, test with behavioral tests)
	@echo "✅ All checks passed!"

# OpenAPI specific targets
openapi-validate: ## Validate OpenAPI specification
	@echo "🔍 Validating OpenAPI specification..."
	@docker run --rm -v $$(pwd):/local openapitools/openapi-generator-cli validate -i /local/openapi.yaml
	@echo "✅ OpenAPI specification is valid!"

openapi-docs: ## Generate OpenAPI documentation
	@echo "📚 Generating OpenAPI documentation..."
	@docker run --rm -v $$(pwd):/local openapitools/openapi-generator-cli generate \
		-i /local/openapi.yaml \
		-g html2 \
		-o /local/docs
	@echo "✅ Documentation generated in ./docs/"

# Quick development workflow
quick: generate build run ## Quick workflow: generate, build, and run

# Full CI/CD workflow  
ci: deps openapi-validate generate format lint test build ## Full CI workflow