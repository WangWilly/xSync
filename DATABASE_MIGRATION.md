# Database Migration Guide

This guide helps you migrate from SQLite to PostgreSQL or configure a new PostgreSQL setup for xSync.

## Quick Setup with Docker

The easiest way to get started with PostgreSQL is using Docker:

```bash
# Start PostgreSQL and pgAdmin
docker-compose -f docker-compose.postgres.yml up -d

# Check if containers are running
docker ps

# View logs
docker-compose -f docker-compose.postgres.yml logs
```

## Configuration

### Option 1: Interactive Configuration

Run the configuration setup:

```bash
./xsync-cli --conf
```

When prompted for database type, enter `postgres` and provide the connection details.

### Option 2: Manual Configuration

Edit your config file (`~/.x_sync/conf.yaml` or `%appdata%/.x_sync/conf.yaml`):

```yaml
root_path: "/path/to/storage"
cookie:
  auth_token: "your_token"
  ct0: "your_ct0"
max_download_routine: 5
database:
  type: "postgres"
  host: "localhost"
  port: "5432"
  user: "xsync"
  password: "xsync_password"
  dbname: "xsync"
```

### Option 3: Environment Variables

For server deployments, you can use environment variables:

```bash
export DB_TYPE=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=xsync
export DB_PASSWORD=xsync_password
export DB_NAME=xsync
```

## Data Migration

Currently, there's no automatic migration tool. If you have existing SQLite data:

1. **Export SQLite data** using a tool like `sqlite3` CLI or DB Browser for SQLite
2. **Import to PostgreSQL** using `psql` or pgAdmin

Example export/import process:

```bash
# Export SQLite data (adjust table names as needed)
sqlite3 xSync.db ".mode csv" ".header on" ".output users.csv" "SELECT * FROM users;"
sqlite3 xSync.db ".mode csv" ".header on" ".output tweets.csv" "SELECT * FROM tweets;"
sqlite3 xSync.db ".mode csv" ".header on" ".output medias.csv" "SELECT * FROM medias;"

# Import to PostgreSQL (after setting up connection)
psql -h localhost -U xsync -d xsync -c "\COPY users FROM 'users.csv' WITH CSV HEADER;"
psql -h localhost -U xsync -d xsync -c "\COPY tweets FROM 'tweets.csv' WITH CSV HEADER;"
psql -h localhost -U xsync -d xsync -c "\COPY medias FROM 'medias.csv' WITH CSV HEADER;"
```

## Benefits of PostgreSQL

- **Better Performance**: Especially for large datasets and concurrent access
- **Advanced Features**: Full-text search, JSON support, advanced indexing
- **Scalability**: Better handling of multiple users and large datasets
- **Backup & Recovery**: More robust backup and point-in-time recovery options
- **Monitoring**: Better tools for monitoring and performance tuning

## Troubleshooting

### Connection Issues

1. **Check PostgreSQL is running**:
   ```bash
   docker ps  # If using Docker
   # or
   systemctl status postgresql  # If installed directly
   ```

2. **Verify connection settings**:
   ```bash
   psql -h localhost -U xsync -d xsync -c "SELECT version();"
   ```

3. **Check firewall/network settings** if connecting to a remote PostgreSQL server

### Schema Issues

If you encounter schema-related errors, the application will automatically create the necessary tables on first connection.

### Performance Tips

1. **Configure PostgreSQL** for your workload in `postgresql.conf`
2. **Monitor query performance** using pgAdmin or `pg_stat_statements`
3. **Regular maintenance**: Use `VACUUM` and `ANALYZE` for optimal performance

## Switching Back to SQLite

To switch back to SQLite, update your configuration:

```yaml
database:
  type: "sqlite"
  path: "/path/to/storage/data/xSync.db"
```

Or set environment variable:
```bash
export DB_TYPE=sqlite
export DB_PATH=./data/xSync.db
```
