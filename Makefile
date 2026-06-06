.PHONY: help build run test clean docker-build docker-run docker-stop install-deps fmt lint

help:
	@echo "Link Shortener Service - Available Commands"
	@echo ""
	@echo "Development:"
	@echo "  make install-deps    Install Go dependencies"
	@echo "  make run             Run the application (requires DATABASE_URL)"
	@echo "  make build           Build the binary"
	@echo "  make test            Run tests"
	@echo "  make fmt             Format code"
	@echo "  make lint            Run linter"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build    Build Docker image"
	@echo "  make docker-run      Start Docker containers (docker-compose)"
	@echo "  make docker-stop     Stop Docker containers"
	@echo "  make docker-logs     View Docker logs"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean           Remove built binaries"

install-deps:
	go mod download
	go mod tidy

build:
	@echo "Building snap-link..."
	go build -o snap-link .
	@echo "Build complete: ./snap-link"

run:
	@echo "Starting server..."
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "Error: DATABASE_URL environment variable is not set"; \
		exit 1; \
	fi
	go run main.go

dev: install-deps
	@echo "Setting up development environment..."
	@if ! grep -q "DATABASE_URL" .env 2>/dev/null; then \
		echo "Creating .env file from .env.example"; \
		cp .env.example .env; \
		echo "Please configure .env with your database URL"; \
	fi
	@echo "To run the app: make run"

test:
	@echo "Running tests..."
	go test -v ./...

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linter..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install with: brew install golangci-lint"; exit 1; }
	golangci-lint run ./...

docker-build:
	@echo "Building Docker image..."
	docker build -t snap-link:latest .

docker-run:
	@echo "Starting Docker containers..."
	docker-compose up -d
	@echo "Services started:"
	@echo "  App: http://localhost:8080"
	@echo "  PostgreSQL: localhost:5432"
	@echo "  Run 'make docker-logs' to view logs"

docker-stop:
	@echo "Stopping Docker containers..."
	docker-compose down

docker-logs:
	docker-compose logs -f

docker-clean:
	docker-compose down -v
	@echo "Docker containers and volumes removed"

clean:
	@echo "Cleaning up..."
	rm -f snap-link
	go clean
	@echo "Clean complete"

# Environment setup
setup-env:
	@echo "Setting up environment..."
	@if [ ! -f .env ]; then \
		cp .env.example .env; \
		echo ".env file created. Please edit it with your configuration."; \
	else \
		echo ".env file already exists"; \
	fi

# All-in-one development setup
setup: install-deps setup-env
	@echo "Development environment setup complete!"
	@echo "Next steps:"
	@echo "  1. Edit .env with your database configuration"
	@echo "  2. Run: make docker-run  (for PostgreSQL in Docker)"
	@echo "  3. Run: make run         (to start the app)"
	@echo "  4. Test: curl http://localhost:8080/health"

# Production build
prod-build: clean install-deps
	@echo "Building production binary..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o snap-link .
	@echo "Production build complete: ./snap-link"
