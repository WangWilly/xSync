# Troubleshooting Guide for xSync Tweet Dashboard Server

## Common Issues and Solutions

### 1. "Failed to render template: html/template: 'dashboard' is undefined"

**Problem**: The server cannot find or parse the dashboard template.

**Solution**: 
- Make sure you're running from the `/server` directory
- Check that `templates/dashboard.html` exists
- The server has fallback inline templates, so this should work even without external templates

**Command to fix**:
```bash
cd /path/to/your/project/server
make run DB_PATH=/path/to/your/database.db
```

### 2. "Database file not found"

**Problem**: The server cannot find the SQLite database file.

**Solutions**:
- Make sure xSync has been run at least once to create the database
- Check the database path is correct
- Use the full path to the database file

**Common database locations**:
- `/Users/username/Projects/tmd/conf/.data/foo.db`
- `~/.x_sync/tmd.db`
- Check your xSync configuration for the actual path

### 3. "Port already in use"

**Problem**: Another service is using port 8080.

**Solutions**:
```bash
# Use a different port
make run PORT=3000 DB_PATH=/path/to/db.db

# Or find what's using the port
lsof -i :8080

# Kill the process if needed
sudo kill -9 <PID>
```

### 4. Running wrong main.go

**Problem**: Accidentally running the main xSync application instead of the server.

**Solution**: Always run from the `/server` directory:
```bash
cd server
make run DB_PATH=/path/to/database.db
```

### 5. Build errors

**Problem**: Go build fails.

**Solutions**:
```bash
# Update dependencies
go mod tidy

# Clean and rebuild
make clean
make build

# Check Go version (needs 1.21+)
go version
```

## Quick Commands

### Test Configuration
```bash
cd server
make test DB_PATH=/Users/willy.w/Projects/tmd/conf/.data/foo.db
```

### Quick Run with Common Path
```bash
cd server
make quick-run
```

### Manual Run
```bash
cd server
DB_PATH=/Users/willy.w/Projects/tmd/conf/.data/foo.db go run main.go
```

### Build and Run
```bash
cd server
make build
DB_PATH=/Users/willy.w/Projects/tmd/conf/.data/foo.db ./tweet-server
```

## Environment Variables

You can also use environment variables:
```bash
export DB_PATH=/Users/willy.w/Projects/tmd/conf/.data/foo.db
export PORT=8080
cd server
go run main.go
```

## Debugging

### Check Database Contents
```bash
sqlite3 /Users/willy.w/Projects/tmd/conf/.data/foo.db
.tables
.schema users
SELECT COUNT(*) FROM users;
.quit
```

### Check Server Logs
The server outputs logs to the console. Look for:
- "Starting server on port X"
- "Database path: /path/to/db"
- "Open http://localhost:X to view the dashboard"

### Check Template Loading
If templates fail to load, the server will use inline templates. Look for warnings about template loading.

## Success Indicators

When everything works correctly, you should see:
```
2025/07/09 13:27:29 Starting server on port 8080
2025/07/09 13:27:29 Database path: /Users/willy.w/Projects/tmd/conf/.data/foo.db
2025/07/09 13:27:29 Open http://localhost:8080 to view the dashboard
```

Then you can open `http://localhost:8080` in your browser to see the dashboard.

## Getting Help

If you're still having issues:
1. Run `make test` to check configuration
2. Check the full error message
3. Verify file paths and permissions
4. Make sure you're in the correct directory (`/server`)
5. Try the manual commands above
