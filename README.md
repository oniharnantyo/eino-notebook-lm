# Eino Notebook

A production-ready Golang CLI application and HTTP server implementing **Hexagonal Architecture** (Ports and Adapters) for managing notebook operations.

## Features

- 🏗️ **Hexagonal Architecture** - Clean separation of concerns with Domain, Application, Interface, and Infrastructure layers
- 🚀 Built with [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) for powerful CLI capabilities
- ⚙️ Configuration management with `.env` file and environment variable support
- 🌐 HTTP/REST API with [Gorilla Mux](https://github.com/gorilla/mux)
- 📝 Multiple commands: serve, config, version
- 🧪 Unit tests included
- 🎯 Cross-platform support (Linux, macOS, Windows)
- 📦 Ready for production deployment

## Architecture

This project follows **Hexagonal Architecture** principles:

- **Domain Layer** - Business entities, value objects, repository interfaces
- **Application Layer** - Use cases, DTOs, mappers
- **Interface Layer** - HTTP handlers, middleware, routes
- **Infrastructure Layer** - Repository implementations, config, logging

📖 **Documentation:**
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - Architecture overview
- [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) - Development guide with examples

## Installation

### From Source

```bash
git clone https://github.com/oniharnantyo/eino-notebook.git
cd eino-notebook
make install
```

### Using Make

```bash
make build
```

The binary will be created at `bin/eino-notebook`.

## Usage

### Global Commands

```bash
# Show help
eino-notebook --help

# Show version
eino-notebook version

# Use specific env file
eino-notebook --env /path/to/.env <command>

# Set log level
eino-notebook --log-level debug <command>
```

### Commands

#### serve - Start HTTP Server

```bash
# Start server with default settings (localhost:8080)
eino-notebook serve

# Start on custom port
eino-notebook serve --port 9090

# Start on custom host
eino-notebook serve --host 0.0.0.0 --port 8080
```

#### config - Manage Configuration

```bash
# View current configuration
eino-notebook config view

# Initialize a new config file
eino-notebook config init

# Validate configuration
eino-notebook config validate
```

## HTTP API

When you run `eino-notebook serve`, an HTTP server is started with the following endpoints:

### Health & Readiness

```bash
GET /health
GET /ready
```

### Notebooks API

```bash
# Create a notebook
POST /api/v1/notebooks
Content-Type: application/json

{
  "title": "My Notebook",
  "description": "A test notebook",
  "content": "Notebook content here",
  "tags": ["test", "example"]
}

# List notebooks (with pagination and filtering)
GET /api/v1/notebooks?page=1&limit=10&status=active&q=search

# Get a notebook by ID
GET /api/v1/notebooks/{id}

# Update a notebook
PUT /api/v1/notebooks/{id}
Content-Type: application/json

{
  "title": "Updated Notebook",
  "description": "Updated description",
  "content": "Updated content",
  "tags": ["updated"]
}

# Delete a notebook
DELETE /api/v1/notebooks/{id}

# Archive a notebook
POST /api/v1/notebooks/{id}/archive
```

## Configuration

Configuration is loaded from `.env` files in the following locations (in order):

1. File specified via `--env` flag
2. `./.env`
3. `./configs/.env`
4. `$HOME/.env`

Environment variables can be used to override `.env` file values:

- `SERVER_HOST` - Server host (default: localhost)
- `SERVER_PORT` - Server port (default: 8080)
- `LOG_LEVEL` - Log level (debug, info, warn, error)

### Example .env File

```bash
# .env
SERVER_HOST=localhost
SERVER_PORT=8080
LOG_LEVEL=info
```

### Configuration Priority (highest to lowest)

1. Command-line flags (`--port 9090`)
2. System environment variables
3. `.env` file
4. Default values

## Development

### Prerequisites

- Go 1.23 or higher
- Make (optional, for using Makefile)

### Running in Development

```bash
# Run directly
go run main.go serve

# Or use make
make run
```

### Building

```bash
# Build for current platform
make build

# Build for multiple platforms
make build-all

# Install to GOPATH/bin
make install
```

### Testing

```bash
# Run tests
make test

# Run tests with coverage
make test-coverage
```

### Linting and Formatting

```bash
# Format code
make fmt

# Run vet
make vet

# Run linter (requires golangci-lint)
make lint
```

## Project Structure

```
.
├── cmd/                            # CLI commands (Cobra)
│   ├── root.go                    # Root command
│   ├── serve.go                   # Server startup with dependency injection
│   ├── config.go                  # Config management
│   └── version.go                 # Version info
├── internal/
│   ├── core/
│   │   ├── domain/                # Domain Layer
│   │   │   ├── entities/          # Business entities
│   │   │   ├── valueobjects/      # Value objects
│   │   │   ├── errors/            # Domain errors
│   │   │   └── repositories/      # Repository interfaces (ports)
│   │   └── application/           # Application Layer
│   │       ├── usecases/          # Use cases
│   │       ├── dtos/              # Data Transfer Objects
│   │       └── mappers/           # Entity <-> DTO mappers
│   ├── infrastructure/            # Infrastructure Layer
│   │   ├── config/               # Configuration
│   │   ├── persistence/          # Repository implementations
│   │   └── logging/              # Logging
│   └── interfaces/               # Interface Layer
│       └── http/
│           ├── handlers/          # HTTP handlers
│           ├── middleware/        # Middleware
│           └── routes/            # Routes
├── pkg/                           # Public packages
│   ├── logger/                    # Logger
│   ├── uuid/                      # UUID type
│   ├── validator/                 # Validator
│   └── errors/                    # Error types
├── test/
│   ├── unit/                      # Unit tests
│   ├── integration/               # Integration tests
│   └── e2e/                       # E2E tests
├── docs/
│   └── ARCHITECTURE.md            # Architecture documentation
├── .env                           # Environment configuration
├── .env.example                   # Example configuration
├── main.go                        # Entry point
├── Makefile                       # Build automation
├── go.mod                         # Go module definition
└── README.md                      # This file
```

## Adding New Commands

Use cobra-cli to add new commands:

```bash
cobra-cli add <command-name>
```

Example:

```bash
cobra-cli add migrate
```

This creates a new command file in `cmd/` that you can customize.

## Version Information

Version information is embedded at build time using ldflags:

```bash
go build -ldflags "-X main.Version=1.0.0 -X main.Commit=abc123 -X main.BuildDate=2025-01-01T00:00:00Z"
```

## License

[Your License Here]
