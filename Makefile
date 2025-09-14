# Kellogg Music Match - Top-Level Makefile
# Orchestrates backend, UI, and infrastructure deployment

.PHONY: help backend-% ui-% infra-% docker-% dev-% clean-% check status logs

# Default target
help: ## Show this help message
	@echo "🎵 Kellogg Music Match - Full Stack Development & Deployment"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Component-specific commands:"
	@echo "  backend-*     Backend operations (make backend-build, backend-test, etc.)"
	@echo "  ui-*          UI operations (make ui-build, ui-dev, etc.)"
	@echo "  infra-*       Infrastructure operations (make infra-deploy, infra-destroy, etc.)"
	@echo ""

## =============================================================================
## 🏗️  BUILD & DEVELOPMENT
## =============================================================================

build: backend-build ui-build ## Build both backend and UI
	@echo "✅ Full application build complete!"

dev: ## Start full development environment
	@echo "🚀 Starting full development environment..."
	@$(MAKE) backend-dev &
	@sleep 3
	@$(MAKE) ui-dev
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend: http://localhost:8080"

dev-stop: ## Stop development servers
	@echo "🛑 Stopping development servers..."
	@pkill -f "air" || true
	@pkill -f "ng serve" || true
	@pkill -f "npm start" || true
	@echo "✅ Development servers stopped!"

test: backend-test ui-test ## Run all tests
	@echo "✅ All tests complete!"

check: backend-check ui-check ## Run all checks (lint, test, format)
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

docker-clean: ## Clean Docker resources
	@echo "🧹 Cleaning Docker resources..."
	@docker-compose down --rmi all --volumes --remove-orphans 2>/dev/null || true
	@docker system prune -f 2>/dev/null || true
	@echo "✅ Docker cleanup complete!"

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