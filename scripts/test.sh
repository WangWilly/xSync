#!/bin/bash

# Test script for xSync monorepo

set -e

echo "Running tests for xSync..."

# Run tests for all packages
echo "Running all tests..."
go test ./... -v

echo "Tests complete!"
