#!/bin/bash

# Development script to run server application

set -e

DEFAULT_PORT=8080
PORT=${1:-$DEFAULT_PORT}

echo "Starting xSync server on port $PORT..."
go run ./cmd/server/main.go
