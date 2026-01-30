.PHONY: build build-prod clean test lint run-indexer run-server run-web install-tools build-frontend

# Build all binaries
build:
	@echo "Building binaries..."
	@mkdir -p bin
	go build -o bin/docs-rag-indexer cmd/indexer/main.go
	go build -o bin/docs-rag-mcp-server cmd/server/main.go
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
	@echo "Building production binaries..."
	@mkdir -p bin
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/docs-rag-indexer cmd/indexer/main.go
	CGO_ENABLED=1 go build -ldflags="-s -w" -o bin/docs-rag-mcp-server cmd/server/main.go
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

# Run indexer
run-indexer:
	@go run cmd/indexer/main.go

# Run MCP server
run-server:
	@go run cmd/server/main.go

# Run web server (development with hot reload)
run-web-dev:
	@echo "Starting frontend dev server..."
	cd web && npm run dev &
	@echo "Starting backend..."
	go run cmd/web/main.go --ollama

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
	@echo "  build          - Build all binaries"
	@echo "  build-prod     - Build production binaries with optimizations"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage"
	@echo "  bench          - Run benchmarks"
	@echo "  lint           - Run linters"
	@echo "  install-tools  - Install development tools"
	@echo "  run-indexer    - Run indexer (development)"
	@echo "  run-server     - Run MCP server (development)"
	@echo "  run-web        - Run web server (development)"
	@echo "  init-db        - Initialize SQLite database"
	@echo "  help           - Show this help message"
