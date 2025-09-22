# Kellogg Music Match - Top-Level Makefile
# Orchestrates backend, UI, and infrastructure deployment

.PHONY: help backend-% ui-% infra-% docker-% dev-db-re	@ecdb-reset: ## Reset database with fresh schema from Flyway migrations
	@echo "🔄 Resetting database with Flyway migrations..."
	@echo "🛑 Step 1: Stopping containers..."
	@docker-compose down
	@echo "🗑️  Step 2: Removing postgres volume to force reinitialization..."
	@docker volume rm kellogg-music-match_postgres_data 2>/dev/null || true
	@echo "🚀 Step 3: Starting postgres..."
	@docker-compose up -d postgres
	@echo "⏳ Step 4: Waiting for database to be ready..."
	@sleep 10
	@echo "📋 Step 5: Running Flyway migrations..."
	@./scripts/flyway.sh migrate
	@echo "✅ Database reset complete with latest migrations!"

db-migrate: ## Apply pending Flyway migrations
	@echo "📋 Applying Flyway migrations..."
	@./scripts/flyway.sh migrate
	@echo "✅ Migrations applied!"

db-info: ## Show Flyway migration information
	@echo "📊 Flyway migration status..."
	@./scripts/flyway.sh info

db-clean: ## Clean database schema (Flyway clean)
	@echo "🧹 Cleaning database schema..."
	@./scripts/flyway.sh clean
	@echo "✅ Database schema cleaned!"esetting database with fresh schema..."
	@echo "🔽 Step 1: Stopping containers...""🔽 Step 1: Stopping containers..."
	@docker-compose down
	@echo "🗑️  Step 2: Removing postgres volume to force reinitialization..."
	@docker volume rm kellogg-music-match_postgres_data 2>/dev/null || true
	@echo "🚀 Step 3: Starting postgres with fresh schema..."
	@docker-compose up -d postgres
	@echo "⏳ Step 4: Waiting for database to be ready..." Reset database with fresh schema from migrations
	@echo "🔄 Resetting database with fresh schema..."
	@echo "🔽 Step 1: Stopping containers..."ean-% check status logs

# Default target
help: ## Show this help message
	@echo "🎵 Kellogg Music Match - Full Stack Development & Deployment"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Component-specific commands:"
	@echo "  backend-*             Backend operations (make backend-build, backend-test, etc.)"
	@echo "  ui-*                  UI operations (make ui-build, ui-dev, etc.)"
	@echo "  db-*                  Database operations (make db-start, db-test, etc.)"
	@echo "  infra-*               Infrastructure operations (make infra-deploy, infra-destroy, etc.)"
	@echo ""
	@echo "Testing commands:"
	@echo "  test                  Run all tests (backend, UI, database)"
	@echo "  test-behavioral       Run Ginkgo behavioral tests"
	@echo "  test-quick           Run quick unit tests only"
	@echo ""
	@echo "Database migrations:"
	@echo "  create-migration      Create new Flyway migration file (make create-migration name=add_feature)"
	@echo "  db-reset              Reset database with guaranteed fresh schema"
	@echo ""
	@echo "Quick development workflow:"
	@echo "  ./dev.sh start        Start full application"
	@echo "  ./dev.sh db-only      Start database only"
	@echo "  ./dev.sh test         Test environment"
	@echo ""

## =============================================================================
## 🏗️  BUILD & DEVELOPMENT
## =============================================================================

build: backend-build ui-build ## Build both backend and UI
	@echo "✅ Full application build complete!"

dev: ## Start full development environment  
	@echo "🚀 Starting full development environment..."
	@echo "📋 Using docker-compose for reliable service management"
	@docker-compose up -d

dev-stop: ## Stop development servers
	@echo "🛑 Stopping development servers..."
	@docker-compose down

test: backend-test ui-test db-test ## Run all tests (backend, UI, database)
	@echo "✅ All tests complete!"

test-behavioral: ## Run backend behavioral tests using Ginkgo
	@echo "🧪 Running behavioral tests..."
	@cd backend && ~/go/bin/ginkgo run ./...
	@echo "✅ Behavioral tests complete!"

test-quick: backend-test-quick ## Run quick tests (backend unit tests only)
	@echo "✅ Quick tests complete!"

check: backend-check ui-check schema-sync test-behavioral ## Run all checks (lint, test, format, schema sync, behavioral tests)
	@echo "✅ All checks passed!"

clean: backend-clean ui-clean docker-clean ## Clean all build artifacts
	@echo "✅ Full cleanup complete!"

## =============================================================================
## 🐳  DOCKER OPERATIONS
## =============================================================================

docker-build: ## Build all Docker images
	@echo "🐳 Building all Docker images..."
	@docker-compose build --parallel
	@echo "✅ All Docker images built!"

docker-build-backend: ## Build backend Docker image only
	@echo "🐳 Building backend Docker image..."
	@docker-compose build backend
	@echo "✅ Backend Docker image built!"

docker-build-ui: ## Build UI Docker image only
	@echo "🐳 Building UI Docker image..."
	@docker-compose build ui
	@echo "✅ UI Docker image built!"

docker-run: docker-build ## Build and start the full application with Docker
	@echo "🐳 Starting full application with Docker..."
	@docker-compose up -d
	@echo "✅ Application started!"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend API: http://localhost:8080"
	@echo "📊 Health Check: http://localhost:8080/health"

docker-stop: ## Stop Docker services
	@echo "🛑 Stopping Docker services..."
	@docker-compose down
	@echo "✅ Docker services stopped!"

docker-restart: docker-stop docker-run ## Restart Docker services

docker-logs: ## Show Docker logs
	@echo "📋 Showing Docker logs..."
	@docker-compose logs -f

docker-db: ## Start PostgreSQL database only
	@echo "🗄️ Starting PostgreSQL database..."
	@docker-compose up -d postgres
	@echo "✅ PostgreSQL database started!"

docker-test: ## Test Docker environment
	@echo "🧪 Testing Docker environment..."
	@./test-docker-compose.sh

docker-clean: ## Clean Docker resources
	@echo "🧹 Cleaning Docker resources..."
	@docker-compose down --rmi all --volumes --remove-orphans 2>/dev/null || true
	@docker system prune -f 2>/dev/null || true
	@echo "✅ Docker cleanup complete!"

logs: ## Show logs for all services
	@echo "📋 Service Logs:"
	@docker-compose logs

## =============================================================================
## �  CONVENIENCE TARGETS
## =============================================================================

create-migration: ## Create a new Flyway migration file (usage: make create-migration name=add_user_roles)
	@if [ -z "$(name)" ]; then \
		echo "❌ Please provide a migration name: make create-migration name=your_migration_name"; \
		exit 1; \
	fi
	@./create-migration.sh "$(name)"

db-start: docker-db ## Start PostgreSQL database only

db-status: ## Show database status
	@echo "📊 Database status..."
	@./dev.sh status

db-logs: ## Show database logs  
	@echo "📋 Database logs..."
	@docker-compose logs postgres

db-connect: ## Connect to database with psql
	@echo "🔗 Connecting to database..."
	@psql -h localhost -p 5432 -U kellogg_user -d kellogg_music_match

db-test: ## Test database setup
	@echo "🧪 Testing database..."
	@./test-docker-compose.sh

db-reset: ## Reset database with fresh schema from migrations
	@echo "🔄 Resetting database with fresh schema..."
	@echo "📋 Step 1: Synchronizing schema files..."
	@echo "� Step 2: Stopping containers..."
	@docker-compose down
	@echo "🗑️  Step 3: Removing postgres volume to force reinitialization..."
	@docker volume rm kellogg-music-match_postgres_data 2>/dev/null || true
	@echo "🚀 Step 4: Starting postgres with fresh schema..."
	@docker-compose up -d postgres
	@echo "⏳ Step 5: Waiting for database to be ready..."
	@sleep 10
	@echo "✅ Database reset complete with latest schema!"

db-backup: ## Create database backup
	@echo "💾 Creating database backup..."
	@mkdir -p backups
	@docker-compose exec -T postgres pg_dump -U kellogg_user kellogg_music_match > backups/backup_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "✅ Backup created in backups/ directory"

db-help: ## Show database commands
	@echo "🗄️ Database Commands:"
	@echo "  db-start              Start PostgreSQL database"
	@echo "  db-status             Show database status"
	@echo "  db-logs               Show database logs"
	@echo "  db-connect            Connect with psql"
	@echo "  db-test               Test database setup"
	@echo "  db-reset              Reset database with Flyway migrations"
	@echo "  db-migrate            Apply pending Flyway migrations"
	@echo "  db-info               Show Flyway migration status"
	@echo "  db-clean              Clean database schema"
	@echo "  create-migration      Create new migration file"
	@echo "  db-schema-verify      Verify database schema matches expected structure"
	@echo "  db-force-reset        Force complete database reset (alias for db-reset)"
	@echo "  db-backup             Create backup"

## =============================================================================
## 🏗️  BACKEND OPERATIONS
## =============================================================================

backend-%: ## Forward commands to backend Makefile
	@echo "🔧 Running backend: $*"
	@cd backend && $(MAKE) $*

## =============================================================================
## 🎨  UI OPERATIONS  
## =============================================================================

ui-build: ## Build UI for production
	@echo "🎨 Building UI for production..."
	@cd ui && npm ci --silent
	@cd ui && npm run build --silent
	@echo "✅ UI build complete!"

ui-dev: ## Start UI development server
	@echo "🎨 Starting UI development server..."
	@cd ui && npm start

ui-test: ## Run UI tests
	@echo "🧪 Running UI tests..."
	@cd ui && npm test -- --watch=false --browsers=ChromeHeadless
	@echo "✅ UI tests complete!"

ui-lint: ## Lint UI code
	@echo "🔍 Linting UI code..."
	@cd ui && npm run lint
	@echo "✅ UI linting complete!"

ui-check: ui-lint ui-test ## Run all UI checks

ui-clean: ## Clean UI build artifacts
	@echo "🧹 Cleaning UI artifacts..."
	@cd ui && rm -rf dist/ node_modules/.cache/ .angular/
	@echo "✅ UI cleanup complete!"

ui-deps: ## Install/update UI dependencies
	@echo "📦 Installing UI dependencies..."
	@cd ui && npm ci
	@echo "✅ UI dependencies installed!"

## =============================================================================
## ☁️  INFRASTRUCTURE OPERATIONS
## =============================================================================

infra-preview: ## Preview infrastructure changes
	@echo "☁️ Previewing infrastructure changes..."
	@cd pulumi && pulumi preview
	@echo "✅ Infrastructure preview complete!"

infra-deploy: ## Deploy infrastructure
	@echo "☁️ Deploying infrastructure..."
	@cd pulumi && pulumi up --yes
	@echo "✅ Infrastructure deployment complete!"

infra-destroy: ## Destroy infrastructure
	@echo "☁️ Destroying infrastructure..."
	@cd pulumi && pulumi destroy --yes
	@echo "✅ Infrastructure destroyed!"

infra-refresh: ## Refresh infrastructure state
	@echo "☁️ Refreshing infrastructure state..."
	@cd pulumi && pulumi refresh --yes
	@echo "✅ Infrastructure refresh complete!"

infra-output: ## Show infrastructure outputs
	@echo "☁️ Infrastructure outputs:"
	@cd pulumi && pulumi stack output

infra-login: ## Login to Pulumi
	@echo "☁️ Logging into Pulumi..."
	@cd pulumi && pulumi login

infra-stack-init: ## Initialize Pulumi stack
	@echo "☁️ Initializing Pulumi stack..."
	@cd pulumi && pulumi stack init dev 2>/dev/null || echo "Stack already exists"

infra-config: ## Configure Pulumi settings
	@echo "☁️ Configuring Pulumi..."
	@cd pulumi && pulumi config set --secret aws:region us-east-1
	@echo "✅ Pulumi configuration complete!"

## =============================================================================
## 📊  MONITORING & STATUS
## =============================================================================

status: ## Show application status
	@echo "📊 Application Status"
	@echo "===================="
	@echo ""
	@echo "🐳 Docker Services:"
	@docker-compose ps 2>/dev/null || echo "  Docker Compose not running"
	@echo ""
	@echo "🌐 Service Health:"
	@curl -s http://localhost:8080/health 2>/dev/null && echo "" || echo "  Backend: ❌ Not responding"
	@curl -s http://localhost:4200 >/dev/null 2>&1 && echo "  Frontend: ✅ Running" || echo "  Frontend: ❌ Not responding"
	@echo ""

logs: docker-logs ## Show application logs

health: ## Check application health
	@echo "🏥 Health Check"
	@echo "==============="
	@echo "Backend Health:"
	@curl -s http://localhost:8080/health | jq 2>/dev/null || curl -s http://localhost:8080/health
	@echo ""

## =============================================================================
## 🚀  DEPLOYMENT WORKFLOWS
## =============================================================================

deploy-local: docker-run ## Deploy locally with Docker
	@echo "✅ Local deployment complete!"

deploy-staging: check docker-build infra-deploy ## Deploy to staging environment
	@echo "🚀 Staging deployment workflow..."
	@# Add staging-specific deployment commands here
	@echo "✅ Staging deployment complete!"

deploy-prod: check docker-build ## Deploy to production environment  
	@echo "🚀 Production deployment workflow..."
	@# Add production-specific deployment commands here
	@echo "⚠️  Production deployment would go here"
	@echo "    Implement with your production deployment strategy"

## =============================================================================
## 🛠️  SETUP & INITIALIZATION
## =============================================================================

setup: ## Initial project setup
	@echo "🛠️ Setting up Kellogg Music Match development environment..."
	@$(MAKE) backend-deps
	@$(MAKE) ui-deps
	@$(MAKE) infra-stack-init
	@echo ""
	@echo "✅ Setup complete! Try these commands:"
	@echo "  make dev         # Start development environment"
	@echo "  make docker-run  # Start with Docker"
	@echo "  make test        # Run all tests"

install-tools: ## Install development tools
	@echo "🔧 Installing development tools..."
	@cd backend && $(MAKE) install-tools
	@echo "✅ Development tools installed!"

## =============================================================================
## 🔄  CI/CD WORKFLOWS
## =============================================================================

ci: ## Full CI workflow
	@echo "🔄 Running full CI workflow..."
	@$(MAKE) check
	@$(MAKE) docker-build
	@$(MAKE) infra-preview
	@echo "✅ CI workflow complete!"

cd-staging: ci deploy-staging ## Full CD workflow for staging

cd-prod: ci deploy-prod ## Full CD workflow for production

## =============================================================================
## 📋  INFORMATION
## =============================================================================

info: ## Show project information
	@echo "🎵 Kellogg Music Match"
	@echo "======================"
	@echo ""
	@echo "📁 Project Structure:"
	@echo "  backend/     Go backend with OpenAPI generation"
	@echo "  ui/          Angular frontend application"  
	@echo "  pulumi/      Infrastructure as Code"
	@echo ""
	@echo "🔗 Development URLs:"
	@echo "  Frontend:    http://localhost:4200"
	@echo "  Backend:     http://localhost:8080"
	@echo "  Health:      http://localhost:8080/health"
	@echo ""
	@echo "🛠️ Available Commands:"
	@echo "  make help    Show all available commands"
	@echo "  make setup   Initial project setup"
	@echo "  make dev     Start development environment"
	@echo "  make docker-run   Start with Docker"
	@echo ""

## =============================================================================
## 🧪  TESTING WORKFLOWS
## =============================================================================

test-unit: backend-test ui-test ## Run unit tests only

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	@$(MAKE) docker-run
	@sleep 5
	@# Add integration test commands here
	@curl -s http://localhost:8080/health >/dev/null && echo "✅ Backend integration test passed"
	@curl -s http://localhost:4200 >/dev/null && echo "✅ Frontend integration test passed"
	@$(MAKE) docker-stop
	@echo "✅ Integration tests complete!"

test-e2e: ## Run end-to-end tests
	@echo "🧪 Running end-to-end tests..."
	@# Add e2e test commands here
	@echo "⚠️  E2E tests would go here"

test-all: test-unit test-integration ## Run all tests