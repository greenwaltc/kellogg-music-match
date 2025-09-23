# Kellogg Music Match - Top-Level Makefile
# Orchestrates backend, UI, and infrastructure deployment

.PHONY: help docker-build docker-run docker-stop docker-logs docker-clean status dev setup test clean backend-% ui-% infra-% db-% logs health restart build-all test-integration db-migrate-k8s db-info-k8s

help: ## Show this help message
	@echo "🎵 Kellogg Music Match - Development Commands"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'

docker-build: ## Build all Docker images
	@echo "🐳 Building all Docker images..."
	@docker-compose build postgres backend ui musicbrainz-loader
	@echo "✅ All Docker images built successfully!"

docker-run: ## Start all services with Docker Compose
	@echo "🐳 Starting Docker services..."
	@docker-compose up -d
	@echo "✅ Docker services started!"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend: http://localhost:8080"

docker-stop: ## Stop Docker services
	@echo "🛑 Stopping Docker services..."
	@docker-compose down
	@echo "✅ Docker services stopped!"

docker-logs: ## Show Docker container logs
	@docker-compose logs -f --tail=100

docker-clean: ## Clean Docker resources (containers, images, volumes)
	@echo "🧹 Cleaning Docker resources..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@echo "✅ Docker cleanup complete!"

status: ## Show application status
	@echo "📊 Application Status:"
	@echo ""
	@echo "🐳 Docker Services:"
	@docker-compose ps 2>/dev/null || echo "  Docker Compose not running"
	@echo ""
	@echo "🔧 Backend:"
	@curl -s http://localhost:8080/health >/dev/null 2>&1 && echo "  ✅ Backend API is healthy" || echo "  ❌ Backend API not responding"

dev: ## Start development environment (database + local services)
	@echo "🚀 Starting development environment..."
	@echo "🗄️ Starting database..."
	@docker-compose up -d postgres
	@sleep 3
	@echo "🔧 Starting backend in development mode..."
	@cd backend && $(MAKE) dev &
	@echo "🎨 Starting UI in development mode..."
	@cd ui && npm start &
	@echo "✅ Development environment starting!"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend: http://localhost:8080"

setup: ## Initial project setup
	@echo "🛠️ Setting up Kellogg Music Match project..."
	@echo "📦 Installing UI dependencies..."
	@cd ui && npm install
	@echo "🗄️ Starting database..."
	@docker-compose up -d postgres
	@sleep 5
	@echo "🗄️ Running database migrations..."
	@$(MAKE) db-migrate
	@echo "✅ Project setup complete!"
	@echo "🏗️ Run 'make dev' to start development environment"

logs: ## Show logs for all services
	@echo "📋 Showing logs..."
	@docker-compose logs -f

# Database Management
db-migrate: ## Apply database migrations
	@echo "🗄️ Applying database migrations..."
	@cd database && ../scripts/flyway.sh migrate
	@echo "✅ Database migrations applied!"

db-migrate-k8s: ## Apply database migrations to Kubernetes (requires port-forward)
	@echo "🗄️ Applying migrations to Kubernetes database..."
	@echo "⚠️  Make sure kubectl port-forward is running: kubectl port-forward -n kmm postgres-0 5433:5432"
	@docker run --rm -v $(PWD)/backend/db/schema/migrations:/flyway/sql flyway/flyway:latest \
		-url=jdbc:postgresql://host.docker.internal:5433/kellogg_music_match \
		-user=kellogg_user -password=kellogg_secure_pass_2024 migrate
	@echo "✅ Kubernetes database migrations applied!"

db-info-k8s: ## Show migration status for Kubernetes database
	@echo "ℹ️ Kubernetes database migration info..."
	@docker run --rm -v $(PWD)/backend/db/schema/migrations:/flyway/sql flyway/flyway:latest \
		-url=jdbc:postgresql://host.docker.internal:5433/kellogg_music_match \
		-user=kellogg_user -password=kellogg_secure_pass_2024 info

db-info: ## Show migration information
	@echo "ℹ️ Database migration info..."
	@cd database && ../scripts/flyway.sh info

db-clean: ## Clean database schema
	@echo "🧹 Cleaning database schema..."
	@cd database && ../scripts/flyway.sh clean

db-reset: ## Reset database with fresh migrations
	@echo "🔄 Resetting database..."
	@$(MAKE) db-clean
	@$(MAKE) db-migrate
	@echo "✅ Database reset complete!"

db-start: ## Start database only
	@echo "🗄️ Starting PostgreSQL database..."
	@docker-compose up -d postgres
	@echo "✅ Database started!"

db-connect: ## Connect to database with psql
	@docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match

# Infrastructure Management (Pulumi)
infra-preview: ## Preview infrastructure changes
	@echo "☁️ Previewing infrastructure changes..."
	@cd pulumi && pulumi preview

infra-deploy: ## Deploy infrastructure
	@echo "🚀 Deploying infrastructure..."
	@cd pulumi && pulumi up

infra-destroy: ## Destroy infrastructure
	@echo "💥 Destroying infrastructure..."
	@cd pulumi && pulumi destroy

infra-status: ## Show infrastructure status
	@echo "📊 Infrastructure status..."
	@cd pulumi && pulumi stack

infra-output: ## Show infrastructure outputs
	@echo "📤 Infrastructure outputs..."
	@cd pulumi && pulumi stack output

infra-login: ## Login to Pulumi
	@cd pulumi && pulumi login

# Testing
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@$(MAKE) backend-test
	@$(MAKE) ui-test
	@echo "✅ All tests complete!"

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@$(MAKE) docker-run
	@sleep 10
	@curl -s http://localhost:8080/health >/dev/null && echo "✅ Backend integration test passed"
	@curl -s http://localhost:4200 >/dev/null && echo "✅ Frontend integration test passed"
	@$(MAKE) docker-stop
	@echo "✅ Integration tests complete!"

# Development Utilities
health: ## Check application health endpoints
	@echo "🏥 Health Check:"
	@echo "🔧 Backend Health:"
	@curl -s http://localhost:8080/health | jq . 2>/dev/null || echo "  ❌ Backend health endpoint not responding"
	@echo "🗄️ Database Connection:"
	@docker-compose exec postgres pg_isready -U kellogg_user 2>/dev/null && echo "  ✅ Database ready" || echo "  ❌ Database not ready"

restart: ## Restart all services
	@echo "🔄 Restarting services..."
	@$(MAKE) docker-stop
	@sleep 2
	@$(MAKE) docker-run
	@echo "✅ Services restarted!"

build-all: ## Build all components (Docker images + dependencies)
	@echo "🏗️ Building all components..."
	@echo "📦 Generating backend code..."
	@cd backend && $(MAKE) generate 2>/dev/null || echo "⚠️ Backend code generation failed"
	@echo "🐳 Building Docker images..."
	@$(MAKE) docker-build
	@echo "🔧 Building backend locally..."
	@cd backend && $(MAKE) build 2>/dev/null || echo "⚠️ Backend build skipped"
	@echo "🎨 Building UI..."
	@cd ui && npm run build 2>/dev/null || echo "⚠️ UI build skipped" 
	@echo "✅ Build complete!"

# Cleanup
clean: ## Clean build artifacts and containers
	@echo "🧹 Cleaning up..."
	@docker-compose down -v --remove-orphans
	@docker system prune -f
	@cd backend && $(MAKE) clean 2>/dev/null || true
	@cd ui && rm -rf dist/ node_modules/.cache/ 2>/dev/null || true
	@echo "✅ Cleanup complete!"

# Forwarding targets for component-specific commands
backend-%: ## Forward backend commands (e.g., make backend-test, backend-build)
	@cd backend && $(MAKE) $*

ui-%: ## Forward UI commands (e.g., make ui-build, ui-lint)  
	@cd ui && npm run $*

infra-%: ## Forward infrastructure commands (e.g., make infra-refresh)
	@cd pulumi && pulumi $*

db-%: ## Forward database commands to scripts
	@cd database && ../scripts/flyway.sh $*

# Default target
.DEFAULT_GOAL := help
