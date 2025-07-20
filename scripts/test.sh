#!/bin/bash

# Test script for xSync monorepo

set -e

echo "Running tests for xSync..."

# Check if we have a specific test target
if [ "$1" == "repos" ]; then
    if [ "$2" == "integration" ]; then
        echo "Running repository tests with integration tests..."
        go test ./pkgs/commonpkg/repos/... -v
    else
        echo "Running repository tests only (skipping integration)..."
        go test ./pkgs/commonpkg/repos/... -short -v
    fi
elif [ "$1" == "repos-integration" ]; then
    echo "Running repository integration tests only..."
    go test ./pkgs/commonpkg/repos/... -run Integration -v
elif [ "$1" == "coverage" ]; then
    echo "Running tests with coverage..."
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out -o coverage.html
    echo "Coverage report generated at coverage.html"
else
    # Run tests for all packages
    echo "Running all tests..."
    go test ./... -short -v
fi

echo "Tests complete!"
