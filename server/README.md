# xSync Tweet Dashboard Server

A web dashboard for viewing recorded tweets and user statistics from the xSync Twitter media downloader.

## Features

- **Dashboard Overview**: View total users, tweets, and media files
- **User Statistics**: See individual user stats, media counts, and activity
- **Tweet Viewer**: Browse tweets that were recorded during download
- **Real-time Updates**: Auto-refresh every 30 seconds
- **Responsive Design**: Works on desktop and mobile devices

## Usage

### Running the Server

```bash
# Navigate to the server directory
cd server

# Run the server (uses default database location)
go run main.go

# Or specify custom database path
DB_PATH=/path/to/your/tmd.db go run main.go

# Or specify custom port
PORT=3000 go run main.go
```

### Environment Variables

- `DB_PATH`: Path to the SQLite database file (default: `~/.x_sync/tmd.db`)
- `PORT`: Server port (default: `8080`)

### Building

```bash
# Build the server
go build -o tweet-server main.go

# Run the built server
./tweet-server
```

## API Endpoints

- `GET /` - Dashboard home page
- `GET /user/{id}` - Get user statistics (JSON)
- `GET /tweets/{entity_id}` - Get tweets for a specific entity (JSON)
- `GET /api/stats` - Get dashboard statistics (JSON)

## Database Schema

The server reads from the following tables:

- `users` - User information (id, screen_name, name, protected, friends_count)
- `user_entities` - User download entities (id, user_id, name, latest_release_time, parent_dir, media_count)
- `user_previous_names` - Historical user names
- `lsts` - Twitter lists
- `lst_entities` - List entities
- `user_links` - User-list relationships

## Development

To add new features:

1. Add new routes in the `Start()` method
2. Create corresponding handler functions
3. Update the HTML template or add new templates
4. Add database queries as needed

## Security Note

This server is intended for local development and monitoring. Do not expose it directly to the internet without proper authentication and security measures.
