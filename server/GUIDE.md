# xSync Tweet Dashboard Server - Complete Guide

## Overview

The xSync Tweet Dashboard Server is a web-based interface for monitoring and viewing Twitter media download statistics from the xSync application. It provides real-time insights into downloaded tweets, user statistics, and media file counts.

## Features

### 🎯 Core Features
- **Real-time Dashboard**: Live statistics and user activity monitoring
- **User Management**: View all tracked users with detailed statistics
- **Tweet Browsing**: Browse and view recorded tweets from failed downloads
- **Media Statistics**: Track media file counts and download progress
- **Activity Timeline**: Monitor user activity and latest download times
- **Responsive Design**: Works seamlessly on desktop and mobile devices

### 📊 Dashboard Components
- **Global Statistics**: Total users, tweets, and media files
- **User Cards**: Individual user statistics and activity
- **Entity Tracking**: Monitor download entities per user
- **Auto-refresh**: Updates every 30 seconds automatically

## Installation & Setup

### Prerequisites
- Go 1.21 or higher
- SQLite3 database from xSync application
- xSync application must be run at least once to create the database

### Quick Start

1. **Clone and Build**
   ```bash
   cd server
   make build
   ```

2. **Run the Server**
   ```bash
   make run
   ```

3. **Access Dashboard**
   Open your browser and navigate to `http://localhost:8080`

### Advanced Setup

#### Custom Configuration
```bash
# Custom port
make run PORT=3000

# Custom database path
make run DB_PATH=/path/to/your/tmd.db

# Both custom port and database
make run PORT=3000 DB_PATH=/path/to/your/tmd.db
```

#### Environment Variables
```bash
export PORT=8080
export DB_PATH=/path/to/tmd.db
./tweet-server
```

#### Using the Start Script
```bash
# Make script executable
chmod +x start.sh

# Run with default settings
./start.sh

# Or with environment variables
PORT=3000 DB_PATH=/custom/path/tmd.db ./start.sh
```

## API Documentation

### Endpoints

#### `GET /`
**Description**: Main dashboard page with complete UI
**Response**: HTML page
**Features**: 
- User statistics overview
- Real-time data updates
- Interactive user cards
- Responsive layout

#### `GET /user/{id}`
**Description**: Get detailed user statistics
**Parameters**: `id` - User ID (uint64)
**Response**: JSON
```json
{
  "User": {
    "Id": 123456789,
    "ScreenName": "username",
    "Name": "Display Name",
    "IsProtected": false,
    "FriendsCount": 500
  },
  "Entities": [...],
  "TotalMedias": 1250,
  "LatestActivity": "2024-01-15T10:30:00Z"
}
```

#### `GET /tweets/{entity_id}`
**Description**: Get tweets for a specific entity
**Parameters**: `entity_id` - Entity ID (int)
**Response**: JSON
```json
{
  "UserName": "username",
  "Tweets": [
    {
      "id": 1234567890,
      "text": "Tweet content...",
      "created_at": "2024-01-15T10:30:00Z",
      "urls": ["https://..."],
      "creator": {...}
    }
  ]
}
```

#### `GET /api/stats`
**Description**: Get dashboard statistics
**Response**: JSON - Same as dashboard data

### Error Handling

All endpoints return appropriate HTTP status codes:
- `200 OK`: Success
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error

## Database Schema

The server reads from the following SQLite tables:

### `users`
- `id` (INTEGER): Unique user identifier
- `screen_name` (VARCHAR): Twitter username
- `name` (VARCHAR): Display name
- `protected` (BOOLEAN): Account protection status
- `friends_count` (INTEGER): Number of accounts followed

### `user_entities`
- `id` (INTEGER): Entity identifier
- `user_id` (INTEGER): Foreign key to users table
- `name` (VARCHAR): Entity name
- `latest_release_time` (DATETIME): Last activity timestamp
- `parent_dir` (VARCHAR): Directory path
- `media_count` (INTEGER): Number of media files

### `user_previous_names`
- `id` (INTEGER): Record identifier
- `uid` (INTEGER): Foreign key to users table
- `screen_name` (VARCHAR): Previous username
- `name` (VARCHAR): Previous display name
- `record_date` (DATE): When the change was recorded

## Development

### Project Structure
```
server/
├── main.go              # Main server application
├── templates/           # HTML templates
│   └── dashboard.html   # Dashboard template
├── static/             # Static assets
│   └── dashboard.css   # Additional CSS
├── go.mod              # Go module definition
├── go.sum              # Go module checksums
├── Makefile            # Build automation
├── start.sh            # Server startup script
├── demo.sh             # Demo and usage guide
└── README.md           # This documentation
```

### Adding New Features

1. **New Routes**: Add routes in the `Start()` method
2. **Handlers**: Create handler functions following the pattern
3. **Templates**: Add HTML templates in `templates/`
4. **Database**: Add database queries in separate methods
5. **API**: Follow RESTful conventions for new endpoints

### Development Mode

For rapid development with auto-reload:
```bash
# Install air for auto-reload
go install github.com/cosmtrek/air@latest

# Run in development mode
make dev
```

## Configuration

### Default Values
- **Port**: 8080
- **Database Path**: `~/.x_sync/tmd.db` (macOS/Linux) or `%APPDATA%/.x_sync/tmd.db` (Windows)
- **Auto-refresh**: 30 seconds
- **Template Path**: `templates/`

### Customization

#### Template Customization
Edit `templates/dashboard.html` to customize the UI:
- Modify CSS in the `<style>` section
- Update HTML structure
- Add JavaScript for new features

#### Database Customization
Create new database methods:
```go
func (s *Server) getCustomData() (interface{}, error) {
    // Custom database query
    var result interface{}
    err := s.db.Select(&result, "SELECT ... FROM ...")
    return result, err
}
```

## Security Considerations

⚠️ **Important**: This server is designed for local development and monitoring.

### Security Notes
- No authentication or authorization
- Direct database access
- No input validation beyond basic type checking
- No rate limiting

### Recommendations for Production
- Add authentication middleware
- Implement proper input validation
- Use HTTPS with TLS certificates
- Add rate limiting and monitoring
- Implement proper error handling and logging
- Use environment-based configuration

## Troubleshooting

### Common Issues

#### "Database file not found"
- Ensure xSync has been run at least once
- Check the database path is correct
- Verify file permissions

#### "Port already in use"
- Change the port with `PORT=3001 make run`
- Check for other services on the port
- Use `lsof -i :8080` to find the process

#### "Template not found"
- Ensure `templates/dashboard.html` exists
- Check file permissions
- Verify the template syntax

#### "Build failed"
- Check Go version (requires 1.21+)
- Run `go mod tidy`
- Ensure all dependencies are available

### Debugging

Enable debug mode:
```bash
# Add debug logging
go run main.go -debug

# Check logs
tail -f ~/.x_sync/server.log
```

## Performance

### Optimization Tips
- Database queries are optimized for the expected data size
- Templates are parsed once at startup
- Static files are served efficiently
- Auto-refresh can be disabled for slower systems

### Monitoring
- Check memory usage with `ps aux | grep tweet-server`
- Monitor database file size
- Watch for slow queries in logs

## Contributing

### Development Setup
1. Fork the repository
2. Create a feature branch
3. Make changes and test thoroughly
4. Submit a pull request

### Code Style
- Follow Go conventions
- Use meaningful variable names
- Add comments for complex logic
- Test all new features

## License

This project is licensed under the same license as the main xSync project.

## Support

For issues, feature requests, or questions:
1. Check this documentation
2. Search existing issues
3. Create a new issue with detailed information
4. Join the Telegram group: https://t.me/+I4yyM81HaJpkNTll

---

**Happy monitoring with xSync Dashboard! 🐦📊**
