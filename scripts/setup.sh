#!/bin/bash

# Setup script for xSync monorepo

set -e

echo "Setting up xSync monorepo..."

# Install dependencies
echo "Installing dependencies..."
go mod tidy
go mod download

# Build applications
echo "Building applications..."
./scripts/build.sh

echo ""
echo "Setup complete!"
echo ""
echo "Available commands:"
echo "  ./bin/xsync-cli --help       - CLI help"
echo "  ./bin/xsync-server           - Start server"
echo "  ./scripts/run-cli.sh --help  - Development CLI"
echo "  ./scripts/run-server.sh      - Development server"
echo "  make help                    - Show all make targets"
echo ""
echo "Quick start:"
echo "  1. Run CLI: ./bin/xsync-cli --help"
echo "  2. Run server: ./bin/xsync-server"
echo "  3. Development mode: ./scripts/dev.sh"
