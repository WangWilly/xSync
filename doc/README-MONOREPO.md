# xSync - Monorepo Structure

This project has been restructured as a monorepo with two main applications:

## Structure

```
xSync/
├── cmd/                    # Main applications
│   ├── cli/               # Command line interface
│   └── server/            # Web server
├── pkgs/                  # Shared packages
├── scripts/               # Build and utility scripts
├── bin/                   # Built binaries (after build)
├── conf/                  # Configuration
├── doc/                   # Documentation
└── ...
```

## Applications

### CLI Application (`cmd/cli`)
The command line interface for downloading X posts.

### Server Application (`cmd/server`)
The web server for viewing downloaded content via a dashboard.

## Quick Start

### Using Scripts

```bash
# Install dependencies
./scripts/install.sh

# Build both applications
./scripts/build.sh

# Run CLI
./scripts/run-cli.sh --help

# Run server
./scripts/run-server.sh [port]

# Run tests
./scripts/test.sh

# Development mode (starts server)
./scripts/dev.sh

# Clean build artifacts
./scripts/clean.sh
```

### Using Make

```bash
# Install dependencies
make install

# Build both applications
make build

# Run CLI (pass arguments with CLI_ARGS)
make run-cli CLI_ARGS="--help"

# Run server (pass port with PORT)
make run-server PORT=8080

# Run tests
make test

# Development mode
make dev

# Clean build artifacts
make clean
```

### Using Go directly

```bash
# CLI
cd cmd/cli
go run . --help

# Server
cd cmd/server
go run . -port=8080
```

## Built Binaries

After running `./scripts/build.sh` or `make build`, binaries are available in `./bin/`:

- `xsync-cli`: Command line interface
- `xsync-server`: Web server

## Script Overview

- `scripts/build.sh`: Build both applications
- `scripts/run-cli.sh`: Run CLI in development mode
- `scripts/run-server.sh`: Run server in development mode
- `scripts/test.sh`: Run all tests
- `scripts/install.sh`: Install dependencies
- `scripts/dev.sh`: Start development server
- `scripts/clean.sh`: Clean build artifacts and caches
