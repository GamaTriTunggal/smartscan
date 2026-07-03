.PHONY: help dev up down build logs db-reset test

help:
	@echo "Smart Label - Development Commands"
	@echo ""
	@echo "Usage:"
	@echo "  make dev        - Start all services in development mode"
	@echo "  make up         - Start all services in background"
	@echo "  make down       - Stop all services"
	@echo "  make build      - Build all containers"
	@echo "  make logs       - View logs"
	@echo "  make db-reset   - Reset database"
	@echo "  make test       - Run tests"
	@echo ""

# Start all services
dev:
	docker-compose up

# Start all services in background
up:
	docker-compose up -d

# Stop all services
down:
	docker-compose down

# Build all containers
build:
	docker-compose build

# View logs
logs:
	docker-compose logs -f

# View backend logs
logs-backend:
	docker-compose logs -f backend

# View frontend logs
logs-frontend:
	docker-compose logs -f frontend

# Reset database
db-reset:
	docker-compose down -v
	docker-compose up -d postgres
	sleep 5
	docker-compose up -d

# Enter backend container
shell-backend:
	docker-compose exec backend sh

# Enter frontend container
shell-frontend:
	docker-compose exec frontend sh

# Enter postgres container
shell-postgres:
	docker-compose exec postgres psql -U smartscan -d smartscan

# Run tests (placeholder)
test:
	@echo "Running tests..."
	cd backend && go test ./...

# Format code
fmt:
	cd backend && go fmt ./...

# Lint code
lint:
	cd backend && golangci-lint run
