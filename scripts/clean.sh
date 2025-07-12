#!/bin/bash

# Clean script for xSync monorepo

set -e

echo "Cleaning xSync build artifacts..."

# Clean binaries
if [ -d "bin" ]; then
    rm -rf bin
    echo "Removed bin directory"
fi

# Clean go mod cache and build cache
go clean -modcache
go clean -cache

# Clean test cache
go clean -testcache

echo "Clean complete!"
