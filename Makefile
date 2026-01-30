.PHONY: build build-prod clean test lint run-web install-tools build-frontend

# Build web server binary
build:
	@echo "Building binary..."
	@mkdir -p bin
	go build -o bin/docs-rag-web cmd/web/main.go
	@echo "Build complete!"

# Build frontend
build-frontend:
	@echo "Building frontend..."
	cd web && npm install && npm run build
	@echo "Frontend build complete!"

# Build all (backend + frontend)
build-all: build build-frontend
	@echo "Full build complete!"

# Build for production with optimizations
build-prod:
	@echo "Building production binary..."
	@mkdir -p bin
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/docs-rag-web cmd/web/main.go
	cd web && npm run build
	@echo "Production build complete!"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -rf data/*.db
	rm -rf data/*.db-*
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Lint code
lint:
	@echo "Running linters..."
	golangci-lint run

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Tools installed!"

# Run web server (development with hot reload)
run-web-dev:
	@echo "Starting frontend dev server..."
	cd web && npm run dev &
	@echo "Starting backend..."
	go run cmd/web/main.go

# Run web server
run-web:
	@go run cmd/web/main.go

# Initialize database
init-db:
	@echo "Initializing database..."
	@mkdir -p data
	sqlite3 data/docs.db < db/schema.sql
	@echo "Database initialized!"

# Help
help:
	@echo "Available targets:"
	@echo "  build          - Build web server binary"
	@echo "  build-frontend - Build React frontend"
	@echo "  build-all      - Build backend + frontend"
	@echo "  build-prod     - Build production binary with optimizations"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linters"
	@echo "  install-tools  - Install development tools"
	@echo "  run-web        - Run web server (development)"
	@echo "  run-web-dev    - Run with frontend hot reload"
	@echo "  init-db        - Initialize SQLite database"
	@echo "  help           - Show this help message"
