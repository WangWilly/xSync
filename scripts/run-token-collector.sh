#!/bin/bash

# Development script to run server application

set -e

echo "Starting xSync token collector..."
go run ./cmd/token-collector/main.go "$@"
