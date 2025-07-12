#!/bin/bash

# Development script to run CLI application

set -e

echo "Running xSync CLI..."
cd cmd/cli
go run . "$@"
