#!/bin/bash

# Development script to start both CLI and server

set -e

echo "Starting xSync development environment..."

# Start server in background
echo "Starting server..."
./scripts/run-server.sh &
SERVER_PID=$!

# Wait a moment for server to start
sleep 2

echo "Server started with PID: $SERVER_PID"
echo "Server should be available at: http://localhost:8080"
echo ""
echo "To use CLI, run: ./scripts/run-cli.sh [options]"
echo "To stop server, run: kill $SERVER_PID"
echo ""
echo "Press Ctrl+C to stop server"

# Handle Ctrl+C
trap "echo 'Stopping server...'; kill $SERVER_PID; exit 0" INT

# Wait for server process
wait $SERVER_PID
