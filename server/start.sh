#!/bin/bash

# xSync Tweet Dashboard Server Launcher
# This script starts the tweet dashboard server

echo "🐦 Starting xSync Tweet Dashboard Server..."

# Set default values
PORT=${PORT:-8080}
DB_PATH=${DB_PATH:-""}

# If DB_PATH is not set, try to find it in the default location
if [ -z "$DB_PATH" ]; then
    if [ -n "$HOME" ]; then
        DEFAULT_DB="$HOME/.x_sync/tmd.db"
    elif [ -n "$APPDATA" ]; then
        DEFAULT_DB="$APPDATA/.x_sync/tmd.db"
    else
        echo "❌ Cannot determine default database location"
        echo "Please set the DB_PATH environment variable"
        exit 1
    fi
    
    if [ -f "$DEFAULT_DB" ]; then
        DB_PATH="$DEFAULT_DB"
        echo "📁 Using database: $DB_PATH"
    else
        echo "❌ Database not found at: $DEFAULT_DB"
        echo "Please run xSync first to create the database, or set DB_PATH environment variable"
        exit 1
    fi
fi

# Check if database exists
if [ ! -f "$DB_PATH" ]; then
    echo "❌ Database file not found: $DB_PATH"
    echo "Please run xSync first to create the database"
    exit 1
fi

echo "🌐 Server will start on port: $PORT"
echo "📊 Dashboard will be available at: http://localhost:$PORT"
echo ""

# Start the server
export DB_PATH
export PORT

go run main.go
