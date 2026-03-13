# Eino Notebook - Hexagonal Architecture

This project implements a production-ready Golang application following **Hexagonal Architecture** (also known as Ports and Adapters architecture).

## Architecture Overview

```
                        EXTERNAL WORLD
                   (Users, Systems, APIs)
                            │
                            ▼
        ┌─────────────────────────────────────┐
        │         PRIMARY ADAPTERS            │
        │      (Interface Layer)              │
        │  ┌─────────────────────────────┐   │
        │  │  HTTP/REST API (Gorilla)    │   │
        │  │  • Handlers                 │   │
        │  │  • Middleware               │   │
        │  │  • Routes                   │   │
        │  └─────────────────────────────┘   │
        │  ┌─────────────────────────────┐   │
        │  │  CLI (Cobra)                │   │
        │  │  • Commands                 │   │
        │  │  • Flags                    │   │
        │  └─────────────────────────────┘   │
        │  ┌─────────────────────────────┐   │
        │  │  gRPC (Future)              │   │
        │  └─────────────────────────────┘   │
        └─────────────────────────────────────┘
                      │
          ┌───────────┴───────────┐
          │   APPLICATION LAYER   │
          │    (Use Cases)        │
          │  ┌─────────────────┐  │
          │  │  NotebookUC     │  │
          │  │  UserUC         │  │
          │  └─────────────────┘  │
          └───────────┬───────────┘
                      │
          ┌───────────┴───────────┐
          │     DOMAIN LAYER      │
          │  (Business Logic)     │
          │  ┌─────────────────┐  │
          │  │  Entities       │  │
          │  │  Value Objects  │  │
          │  │  Repositories   │  │
          │  │  (Ports)        │  │
          │  └─────────────────┘  │
          └───────────┬───────────┘
                      │
                      ▲
          ┌───────────┴───────────┐
          │   SECONDARY ADAPTERS  │
          │  (Infrastructure)      │
          │  ┌─────────────────┐  │
          │  │ PostgreSQL Repo │  │
          │  │  Memory Repo    │  │
          │  │  Redis Cache    │  │
          │  │  External APIs  │  │
          │  └─────────────────┘  │
          └───────────────────────┘
```

## The Hexagon

```
                  ┌─────────────────────────────────────┐
                  │                                     │
                  │         PRIMARY ADAPTERS            │
                  │      (Driving/Inbound)              │
                  │                                     │
          ┌───────┴──────────┬────────────────┬─────────┤
          │                  │                │         │
       HTTP API            CLI             gRPC      GraphQL
          │                  │                │         │
          └──────────────────┴────────────────┴─────────┘
                          │
          ┌───────────────┴───────────────┐
          │                                │
     APPLICATION LAYER              DOMAIN LAYER
    (Orchestrates Logic)           (Core Rules)
          │                                │
          └───────────────┬───────────────┘
                          │
                  ┌───────┴──────────┬────────────────┬─────────┤
          │                  │                │         │
      PostgreSQL         Redis           Message Queue  File System
          │                  │                │         │
          └──────────────────┴────────────────┴─────────┘
                  │
                  │         SECONDARY ADAPTERS
                  │      (Driven/Outbound)
                  │
                  └─────────────────────────────────────┘
```

## Layer Responsibilities

### 1. Domain Layer (`internal/core/domain/`)

The **core** of the application. Contains business logic and is completely independent of external concerns.

```
domain/
├── entities/           # Business entities (Notebook, User)
├── valueobjects/       # Value objects (Status, ID types)
├── errors/            # Domain-specific errors
├── events/            # Domain events
├── repositories/      # Repository interfaces (ports)
└── services/          # Domain services
```

**Key Principles:**
- No external dependencies
- Pure Go business logic
- Interfaces for repositories (ports)
- Domain errors

### 2. Application Layer (`internal/core/application/`)

Orchestrates business logic from the domain layer. Contains use cases that define what the application can do.

```
application/
├── usecases/          # Business logic orchestration
├── dtos/              # Data Transfer Objects
├── mappers/           # Entity <-> DTO mapping
└── ports/             # Input/Output port interfaces
```

**Key Principles:**
- Depends on Domain layer
- Defines use cases (application services)
- Uses DTOs for data transfer
- No infrastructure concerns

### 3. Interface Layer (`internal/interfaces/`)

Handles communication with the outside world (HTTP, gRPC, CLI, etc.).

```
interfaces/
├── http/
│   ├── handlers/      # HTTP request handlers
│   ├── middleware/    # HTTP middleware
│   └── routes/        # Route definitions
└── grpc/              # gRPC handlers (future)
```

**Key Principles:**
- Thin layer - delegates to Application layer
- Handles protocol-specific concerns
- No business logic

### 4. Infrastructure Layer (`internal/infrastructure/`)

Implements interfaces defined in the Domain layer.

```
infrastructure/
├── persistence/       # Repository implementations
├── config/           # Configuration loading
├── logging/          # Logging setup
├── cache/            # Cache implementations
├── messaging/        # Message queue implementations
└── external/         # External service clients
```

**Key Principles:**
- Implements Domain interfaces
- Can be swapped without affecting business logic
- Contains all external dependencies

## Dependency Flow

```
┌──────────────┐
│   Interface  │ ──depends on──>  Application
│   Layer      │ ──depends on──>  Domain
└──────────────┘

┌──────────────┐
│  Infra       │ ──implements──>  Domain (interfaces)
│  Layer       │
└──────────────┘
```

**Rule:** Dependencies point **inward** toward the Domain layer.

## Example: Creating a Notebook

```go
// 1. HTTP Handler (Interface Layer)
func (h *NotebookHandler) Create(w http.ResponseWriter, r *http.Request) {
    var req dtos.CreateNotebookRequest
    json.NewDecoder(r.Body).Decode(&req)

    notebook, err := h.useCase.Create(r.Context(), &req)
    // ...
}

// 2. Use Case (Application Layer)
func (uc *notebookUseCase) Create(ctx context.Context, req *dtos.CreateNotebookRequest) (*dtos.NotebookResponse, error) {
    notebook, err := entities.NewNotebook(req.Title, req.Description, req.Content, req.Tags)
    if err := uc.notebookRepo.Save(ctx, notebook); err != nil {
        return nil, err
    }
    return mappers.ToNotebookResponse(notebook), nil
}

// 3. Entity (Domain Layer)
func NewNotebook(title, description, content string, tags []string) (*Notebook, error) {
    notebook := &Notebook{
        ID: uuid.New(),
        Title: title,
        // ...
    }
    return notebook, notebook.Validate()
}

// 4. Repository Implementation (Infrastructure Layer)
func (r *InMemoryNotebookRepository) Save(ctx context.Context, notebook *entities.Notebook) error {
    r.mu.Lock()
    r.nbs[notebook.ID] = notebook
    r.mu.Unlock()
    return nil
}
```

## Project Structure

```
.
├── cmd/                          # CLI commands (Cobra)
│   ├── root.go                  # Root command
│   ├── serve.go                 # Server startup with DI
│   ├── config.go                # Config management
│   └── version.go               # Version info
├── internal/
│   └── core/
│       ├── domain/              # Domain layer
│       │   ├── entities/        # Business entities
│       │   ├── valueobjects/    # Value objects
│       │   ├── errors/          # Domain errors
│       │   └── repositories/    # Repository interfaces
│       └── application/         # Application layer
│           ├── usecases/        # Use cases
│           ├── dtos/            # DTOs
│           └── mappers/         # Mappers
│   ├── infrastructure/          # Infrastructure layer
│   │   ├── config/             # Configuration
│   │   ├── persistence/        # Repository implementations
│   │   └── logging/            # Logging
│   └── interfaces/             # Interface layer
│       └── http/
│           ├── handlers/        # HTTP handlers
│           ├── middleware/      # Middleware
│           └── routes/          # Routes
├── pkg/                         # Public packages
│   ├── logger/                  # Logger
│   ├── uuid/                    # UUID type
│   ├── validator/               # Validator
│   └── errors/                  # Error types
├── test/
│   ├── unit/                    # Unit tests
│   ├── integration/             # Integration tests
│   └── e2e/                     # E2E tests
├── .env                         # Environment configuration
├── Makefile                     # Build automation
└── README.md                    # Documentation
```

## Benefits of This Architecture

1. **Testability:** Each layer can be tested in isolation with mocks
2. **Flexibility:** Infrastructure can be swapped (e.g., PostgreSQL → MongoDB)
3. **Maintainability:** Clear separation of concerns
4. **Scalability:** Easy to add new interfaces (CLI, gRPC, etc.)
5. **Domain-Driven:** Business logic is isolated and protected

## Adding New Features

1. **Domain:** Add entity and repository interface
2. **Application:** Add use case and DTOs
3. **Infrastructure:** Implement repository
4. **Interface:** Add HTTP/gRPC handlers
5. **Wire it up:** Update serve.go with dependency injection

## Configuration

Configuration via `.env` file:

```bash
# .env
SERVER_HOST=localhost
SERVER_PORT=8080
LOG_LEVEL=info
```

Priority: CLI flags > Environment variables > .env file > Defaults
