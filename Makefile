# Makefile for xSync monorepo

.PHONY: build clean test test-repos test-repos-integration test-integration-only test-coverage install dev run-cli run-server setup help

# Default target
help:
	@echo "xSync Monorepo Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  setup             - Setup project (install deps + build)"
	@echo "  build             - Build both CLI and server applications"
	@echo "  clean             - Clean build artifacts and caches"
	@echo "  test              - Run all tests (skips integration tests)"
	@echo "  test-repos        - Run only repository tests (skips integration tests)"
	@echo "  test-repos-integration - Run repository tests including integration tests"
	@echo "  test-integration-only - Run only integration tests"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  install           - Install dependencies"
	@echo "  dev               - Start development server"
	@echo "  run-cli           - Run CLI application (pass args with CLI_ARGS=...)"
	@echo "  run-server        - Run server application (pass port with PORT=...)"
	@echo "  help              - Show this help message"

build:
	@./scripts/build.sh

clean:
	@./scripts/clean.sh

test:
	@./scripts/test.sh

test-repos:
	@./scripts/test.sh repos

test-repos-integration:
	@./scripts/test.sh repos integration

test-integration-only:
	@./scripts/test.sh repos-integration

test-coverage:
	@./scripts/test.sh coverage

install:
	@./scripts/install.sh

dev:
	@./scripts/dev.sh

run-cli:
	@./scripts/run-cli.sh $(CLI_ARGS)

run-server:
	@./scripts/run-server.sh $(PORT)

setup:
	@./scripts/setup.sh
