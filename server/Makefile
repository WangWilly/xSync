# Makefile for xSync Tweet Dashboard Server

.PHONY: build clean run dev test quick-run install demo help

# Variables
BINARY_NAME=tweet-server
PORT=8080
DB_PATH=

# Default target
all: build

# Build the server
build:
	@echo "🔧 Building xSync Tweet Dashboard Server..."
	go build -o $(BINARY_NAME) main.go
	@echo "✅ Build complete!"

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	@echo "✅ Clean complete!"

# Run the server (builds first if needed)
run: build
	@echo "🚀 Starting xSync Tweet Dashboard Server..."
	@if [ -n "$(DB_PATH)" ]; then \
		echo "📁 Using database: $(DB_PATH)"; \
		DB_PATH="$(DB_PATH)" PORT=$(PORT) ./$(BINARY_NAME); \
	else \
		echo "🔍 Auto-detecting database location..."; \
		PORT=$(PORT) ./$(BINARY_NAME); \
	fi

# Development mode with auto-reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	@echo "🔥 Starting development server with auto-reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "❌ Air not found. Install it with: go install github.com/cosmtrek/air@latest"; \
		echo "Or use 'make run' for regular execution"; \
	fi

# Test the server without building
test:
	@echo "🧪 Testing server configuration..."
	@echo "Working directory: $$(pwd)"
	@echo "Go version: $$(go version)"
	@if [ -n "$(DB_PATH)" ]; then \
		echo "Database path: $(DB_PATH)"; \
		if [ -f "$(DB_PATH)" ]; then \
			echo "✅ Database file exists"; \
		else \
			echo "❌ Database file not found"; \
		fi; \
	else \
		echo "No database path specified, will auto-detect"; \
	fi
	@echo "Port: $(PORT)"

# Quick run with common database path
quick-run:
	@echo "🏃 Quick run with common database path..."
	@make run DB_PATH=/Users/willy.w/Projects/tmd/conf/.data/foo.db

# Install the server globally
install: build
	@echo "📦 Installing xSync Tweet Dashboard Server..."
	cp $(BINARY_NAME) /usr/local/bin/
	@echo "✅ Server installed to /usr/local/bin/$(BINARY_NAME)"

# Show demo information
demo:
	@./demo.sh

# Show help
help:
	@echo "xSync Tweet Dashboard Server - Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build     - Build the server binary"
	@echo "  clean     - Remove build artifacts"
	@echo "  run       - Build and run the server"
	@echo "  dev       - Run in development mode with auto-reload"
	@echo "  test      - Test server configuration"
	@echo "  quick-run - Quick run with common database path"
	@echo "  install   - Install the server globally"
	@echo "  demo      - Show demo information"
	@echo "  help      - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  PORT      - Server port (default: 8080)"
	@echo "  DB_PATH   - Database path (default: auto-detect)"
	@echo ""
	@echo "Examples:"
	@echo "  make run"
	@echo "  make run PORT=3000"
	@echo "  make run DB_PATH=/path/to/db.sqlite"
	@echo "  make run PORT=3000 DB_PATH=/path/to/db.sqlite"
