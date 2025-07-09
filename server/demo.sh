#!/bin/bash

# Demo script for xSync Tweet Dashboard Server

echo "=== xSync Tweet Dashboard Server Demo ==="
echo ""

echo "🔧 Setting up..."
echo "Building the server..."
go build -o tweet-server main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed!"
    exit 1
fi

echo "✅ Server built successfully!"
echo ""

echo "📋 Usage Instructions:"
echo "1. Make sure you have run xSync at least once to create the database"
echo "2. The database should be located at: ~/.x_sync/tmd.db (macOS/Linux) or %APPDATA%/.x_sync/tmd.db (Windows)"
echo "3. Run the server with one of these commands:"
echo ""
echo "   # Use default settings (port 8080, auto-detect database)"
echo "   ./tweet-server"
echo ""
echo "   # Use custom port"
echo "   PORT=3000 ./tweet-server"
echo ""
echo "   # Use custom database path"
echo "   DB_PATH=/path/to/your/tmd.db ./tweet-server"
echo ""
echo "   # Use both custom port and database path"
echo "   PORT=3000 DB_PATH=/path/to/your/tmd.db ./tweet-server"
echo ""
echo "4. Open your browser and navigate to: http://localhost:8080 (or your custom port)"
echo ""

echo "📊 Features:"
echo "- Dashboard with user statistics"
echo "- Real-time data updates"
echo "- Tweet viewing and browsing"
echo "- Media file counts"
echo "- User activity timeline"
echo "- Responsive design for mobile and desktop"
echo ""

echo "🛠️ API Endpoints:"
echo "- GET /                    - Main dashboard"
echo "- GET /user/{id}           - User details (JSON)"
echo "- GET /tweets/{entity_id}  - User tweets (JSON)"
echo "- GET /api/stats           - Dashboard stats (JSON)"
echo ""

echo "🚀 Quick Start:"
echo "If you want to start the server now, run:"
echo "  ./start.sh"
echo ""

echo "📝 Note: The server will auto-refresh every 30 seconds to show latest data."
echo "You can also manually refresh by clicking the refresh button."
echo ""
echo "🎉 Happy monitoring!"
