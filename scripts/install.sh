#!/bin/bash

# Install dependencies for xSync monorepo

set -e

echo "Installing dependencies for xSync..."

# Install dependencies
go mod tidy
go mod download

echo "Dependencies installed!"
