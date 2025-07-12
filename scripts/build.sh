#!/bin/bash

# Build script for xSync monorepo

set -e

echo "Building xSync applications..."

# Build CLI application
echo "Building CLI application..."
cd cmd/cli
go build -o ../../bin/xsync-cli .
cd ../..

# Build server application
echo "Building server application..."
cd cmd/server
go build -o ../../bin/xsync-server .
cd ../..

echo "Build complete!"
echo "Binaries available in ./bin/"
echo "  - xsync-cli: Command line interface"
echo "  - xsync-server: Web server"
