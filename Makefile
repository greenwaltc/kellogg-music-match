# Affyne - Development Commands

# Image versioning
IMAGE_TAG ?= $(shell date +%Y%m%d-%H%M%S)
GIT_SHA ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
IMAGE_VERSION ?= $(IMAGE_TAG)-$(GIT_SHA)

.PHONY: help docs dev status events-status events-sample clean backend-test build docker-build docker-run docker-stop docker-logs docker-clean test infra-preview infra-deploy infra-destroy k3s-import k3s-build-import k3s-deploy k3s-status db-migrate db-connect generate-vapid-keys

help:
	@echo "🎵 Affyne - Available Commands:"
	@echo ""
	@echo "📋 General:"
	@echo "  make help           Show this help"
	@echo "  make docs           Documentation navigation"
	@echo "  make dev            Start development environment"
	@echo "  make status         Application status"
	@echo ""
	@echo "🏗️ Building:"
	@echo "  make build          Build all components"
	@echo "  make docker-build   Build Docker images"
	@echo "  make test           Run all tests"
	@echo ""
	@echo "🐳 Docker:"
	@echo "  make docker-run     Start all services"
	@echo "  make docker-run-debug  Start all services with backend debug"
	@echo "  make docker-stop    Stop all services"
	@echo "  make docker-logs    Show container logs"
	@echo "  make docker-clean   Clean containers and images"
	@echo ""
	@echo "🎵 Chicago Events:"
	@echo "  make events-status  Show event counts"
	@echo "  make events-sample  Show sample events"
	@echo ""
	@echo "🗄️ Database:"
	@echo "  make db-migrate     Apply migrations"
	@echo "  make db-connect     Connect to database"
	@echo ""
	@echo "☁️ Infrastructure:"
	@echo "  make infra-preview  Preview infra changes"
	@echo "  make infra-deploy   Deploy infrastructure"
	@echo "  make infra-destroy  Destroy infrastructure"
	@echo ""
	@echo "🚢 k3s:"
	@echo "  make k3s-import     Import images to k3s"
	@echo "  make k3s-deploy     Deploy to k3s"
	@echo "  make k3s-status     Show k3s status"
	@echo ""
	@echo "🧹 Cleanup:"
	@echo "  make clean          Clean containers"
	@echo ""
	@echo "🔐 Web Push / VAPID:"
	@echo "  make generate-vapid-keys  Generate a new VAPID key pair and print .env lines"

docs:
	@echo "📚 Documentation Navigation:"
	@echo ""
	@echo "🚀 Quick Start:"
	@echo "  README.md - Complete setup and API guide"  
	@echo "  make dev  - Start development environment"
	@echo ""
	@echo "📖 Full Documentation:"
	@echo "  DOCS.md - Complete documentation index"
	@echo "  TICKETMASTER_INTEGRATION.md - Chicago Events & API"
	@echo "  DATABASE.md - Database setup and schema"

dev:
	@echo "🚀 Starting development environment..."
	docker-compose up -d
	@echo "✅ Services started!"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend: http://localhost:8080"
	@echo "📍 Chicago Events: http://localhost:8080/chicago/events"

status:
	@echo "�� Application Status:"
	@docker-compose ps || echo "Docker not running"
	@curl -s http://localhost:8080/health >/dev/null && echo "✅ Backend healthy" || echo "❌ Backend down"

events-status:
	@echo "�� Chicago Events Status:"
	@docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match -c "SELECT COUNT(*) FROM concert_events;" || echo "❌ Database not accessible"

events-sample:
	@echo "🎵 Sample Chicago Events:"
	@curl -s "http://localhost:8080/chicago/events?limit=3" | jq -r '.events[].name' || echo "❌ Backend not accessible"

clean:
	@echo "🧹 Cleaning up..."
	docker-compose down -v
	@echo "✅ Cleanup complete!"

backend-test:
	@cd backend && make test

# Building Commands
build: ## Build all components
	@echo "🏗️ Building all components..."
	@cd backend && make generate
	@cd backend && make generate-sqlc
	@echo "✅ Build complete!"

# Docker Commands
docker-build: ## Build all Docker images
	@echo "🐳 Building all Docker images..."
	docker-compose build postgres backend ui musicbrainz-loader
	@echo "✅ All Docker images built!"

docker-run: ## Start all services with Docker Compose
	@echo "🐳 Starting Docker services..."
	docker-compose up -d
	@echo "✅ Docker services started!"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend: http://localhost:8080"

docker-run-debug: ## Start services with backend in Delve debug mode
	@echo "🐞 Starting Docker services (debug backend)..."
	docker compose -f docker-compose.yml -f docker-compose.debug.yml up -d --build
	@echo "✅ Services started in debug mode"
	@echo "🌐 Frontend: http://localhost:4200"
	@echo "🔧 Backend: http://localhost:8080"
	@echo "🪲 Delve:    127.0.0.1:2345"

docker-stop: ## Stop Docker services
	@echo "🛑 Stopping Docker services..."
	docker-compose down
	@echo "✅ Docker services stopped!"

docker-logs: ## Show Docker container logs
	docker-compose logs -f --tail=100

docker-clean: ## Clean Docker resources
	@echo "🧹 Cleaning Docker resources..."
	docker-compose down -v --remove-orphans
	docker system prune -f
	@echo "✅ Docker cleanup complete!"

# Testing Commands
test: ## Run all tests
	@echo "🧪 Running all tests..."
	@make backend-test
	@echo "✅ All tests complete!"

# Database Commands
db-migrate: ## Apply database migrations
	@echo "🗄️ Applying database migrations..."
	@cd database && ../scripts/flyway.sh migrate
	@echo "✅ Database migrations applied!"

db-connect: ## Connect to database with psql
	docker-compose exec postgres psql -U kellogg_user -d kellogg_music_match

# Infrastructure Commands (Pulumi)
infra-preview: ## Preview infrastructure changes
	@echo "☁️ Previewing infrastructure changes..."
	@cd pulumi && pulumi preview

infra-deploy: docker-build ## Build images and deploy infrastructure
	@echo "🚀 Deploying infrastructure..."
	@cd pulumi && pulumi up

infra-destroy: ## Destroy infrastructure
	@echo "💥 Destroying infrastructure..."
	@cd pulumi && pulumi destroy

# k3s Commands
k3s-import: ## Import Docker images to k3s
	@echo "🚢 Importing Docker images to k3s..."
	@./scripts/k3s-image-import.sh import
	@echo "✅ Images imported to k3s!"

k3s-build-import: docker-build k3s-import ## Build and import images to k3s

k3s-deploy: k3s-build-import ## Deploy to k3s
	@echo "🚀 Deploying to k3s..."
	@kubectl apply -f pulumi/k8s/ -n affyne || echo "⚠️ Manual k8s deployment needed"
	@echo "✅ k3s deployment complete!"

k3s-status: ## Show k3s cluster status
	@echo "📊 k3s Cluster Status:"
	@echo ""
	@echo "🏗️ Nodes:"
	@sudo k3s kubectl get nodes
	@echo ""
	@echo "🚀 Pods:"
	@kubectl get pods -n affyne 2>/dev/null || echo "  No pods in affyne namespace"
	@echo ""
	@echo "🔧 Services:"
	@kubectl get services -n affyne 2>/dev/null || echo "  No services in affyne namespace"

# Forwarding Commands
backend-%: ## Forward backend commands
	@cd backend && make $*

ui-%: ## Forward UI commands
	@cd ui && npm run $*

infra-%: ## Forward infrastructure commands
	@cd pulumi && pulumi $*

ngrok-tunnel: ## Start ngrok tunnel for backend
	@echo "🌐 Starting ngrok tunnel..."
	ngrok http 4200
	@echo "✅ ngrok tunnel started!"

.DEFAULT_GOAL := help

generate-vapid-keys:
	@set -e; \
	if command -v npx >/dev/null 2>&1; then \
	  CMD="npx --yes web-push generate-vapid-keys"; \
	else \
	  echo "[vapid] npx not found; falling back to Docker (node:20-alpine)"; \
	  CMD="docker run --rm node:20-alpine sh -lc 'npx --yes web-push generate-vapid-keys'"; \
	fi; \
	OUT=$$(sh -lc "$$CMD"); \
	echo "$$OUT"; \
	PUB=$$(printf "%s\n" "$$OUT" | awk -F': ' '/Public Key/{print $$2}'); \
	PRIV=$$(printf "%s\n" "$$OUT" | awk -F': ' '/Private Key/{print $$2}'); \
	echo ""; \
	echo "🔐 Copy-paste into your .env (or export in shell):"; \
	echo "VAPID_PUBLIC_KEY=$$PUB"; \
	echo "VAPID_PRIVATE_KEY=$$PRIV"; \
	echo "VAPID_SUBJECT=mailto:support@kelloggmatch.com"; \
	echo ""; \
	echo "Tip: update docker-compose and restart: docker compose up -d --build"

run-iphone-17:
	xcrun simctl boot D67581D4-2783-4A81-BA85-B67358FB94AA

flutter-run-ngrok: ## Run Flutter app with API base set to current ngrok URL
	@cd ui_flutter && flutter run -d ios \
	  --dart-define=API_BASE_URL=https://twolegged-leonida-interchangeably.ngrok-free.dev