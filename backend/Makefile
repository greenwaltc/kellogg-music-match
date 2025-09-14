# Makefile for Kellogg Music Match Backend
# This Makefile provides targets for OpenAPI code generation and common development tasks

.PHONY: help generate clean build test run docker-build docker-run docker-stop dev

# Default target
help: ## Show this help message
	@echo "Kellogg Music Match Backend - Available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""

generate: ## Generate OpenAPI server code from openapi.yaml
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

clean: ## Clean build artifacts and temporary files
	@echo "🧹 Cleaning build artifacts..."
	@rm -f server backend
	@rm -rf generated/temp_gen generated/temp_backup
	@rm -f *.log
	@go clean
	@echo "✅ Clean complete!"

build: ## Build the backend server
	@echo "🔨 Building backend server..."
	@go mod tidy
	@go build -o server ./cmd
	@echo "✅ Build complete! Binary: ./server"

test: ## Run tests
	@echo "🧪 Running tests..."
	@go test ./... -v
	@echo "✅ Tests complete!"

run: build ## Build and run the server locally
	@echo "🚀 Starting server on :8080..."
	@./server

dev: ## Start development server with live reload (requires 'air' tool)
	@if command -v air >/dev/null 2>&1; then \
		echo "🔥 Starting development server with live reload..."; \
		air; \
	else \
		echo "❌ 'air' tool not found. Install with: go install github.com/cosmtrek/air@latest"; \
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
	@go install github.com/cosmtrek/air@latest
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

check: clean lint test ## Run full check (clean, lint, test)
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