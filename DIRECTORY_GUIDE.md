# Directory Structure Guide

This project follows **Hexagonal (Clean) Architecture** with clear separation of concerns.

## Root Directories

### `/cmd` — Application Entry Points
**Purpose:** Main application binaries and CLI commands.

| File | Purpose |
|------|---------|
| `root.go` | Root Cobra command setup |
| `serve.go` | HTTP server command |
| `config.go` | Config-related commands |
| `version.go` | Version information |

**Guidelines:**
- Keep code minimal - only wire dependencies together
- No business logic here
- Each file represents a different binary/command

---

### `/internal` — Private Application Code
**Purpose:** Application code that should not be imported by external projects.

**Structure:** Organized into three layers:
1. **Domain** (core business rules)
2. **Application** (use cases)
3. **Infrastructure** (external concerns)
4. **Interfaces** (adapters)

---

## `/internal/core` — Core Business Logic

### `/internal/core/domain` — Domain Layer
**Purpose:** Enterprise business rules and entities. The heart of your application.

#### `/entities/`
**Purpose:** Core business entities with behavior.

```go
// Example: Notebook, Document
type Notebook struct {
    ID        uuid.UUID
    Title     string
    Status    NotebookStatus
    // ...
}

func (n *Notebook) Validate() error { ... }
func (n *Notebook) Archive() { ... }
```

**Guidelines:**
- Entities should be self-contained
- Include business logic methods
- Use value objects for complex attributes
- No framework dependencies

#### `/valueobjects/`
**Purpose:** Immutable types that represent domain concepts.

```go
// Example: NotebookStatus, Email, UserID
type NotebookStatus string

const (
    StatusActive   NotebookStatus = "active"
    StatusArchived NotebookStatus = "archived"
)
```

**Guidelines:**
- Use for types with validation rules
- Make immutable when possible
- Define related constants here

#### `/repositories/`
**Purpose:** Interfaces for data persistence (ports).

```go
type NotebookRepository interface {
    Save(ctx context.Context, notebook *entities.Notebook) error
    FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error)
    // ...
}
```

**Guidelines:**
- Define interfaces only (no implementations)
- Return domain entities, not DTOs
- Use context.Context for cancellation
- Keep methods focused on domain needs

#### `/errors/`
**Purpose:** Domain-specific error types.

```go
var ErrEmptyTitle = errors.New("title cannot be empty")
var ErrTitleTooLong = errors.New("title exceeds maximum length")
```

**Guidelines:**
- Define domain error constants
- Create error constructors when needed
- Don't use HTTP status codes here

#### `/services/` (optional)
**Purpose:** Domain services for complex business logic that doesn't fit in entities.

```go
// Example: DocumentIndexingService
type DocumentIndexingService interface {
    IndexDocument(ctx context.Context, doc *Document) error
}
```

---

### `/internal/core/application` — Application Layer
**Purpose:** Orchestrate domain objects to perform use cases.

#### `/usecases/`
**Purpose:** Implement business workflows and application rules.

```go
type NotebookUseCase interface {
    Create(ctx context.Context, req *dtos.CreateNotebookRequest) (*dtos.NotebookResponse, error)
    GetByID(ctx context.Context, id string) (*dtos.NotebookResponse, error)
}

type notebookUseCase struct {
    notebookRepo repositories.NotebookRepository
}
```

**Guidelines:**
- Define interface first, then implement
- Use dependency injection (accept repositories via constructor)
- Handle transaction boundaries
- Return DTOs, not domain entities
- Don't expose domain details to interfaces

#### `/dtos/`
**Purpose:** Data Transfer Objects for external communication.

```go
type CreateNotebookRequest struct {
    Title       string   `json:"title" validate:"required"`
    Description string   `json:"description"`
    Tags        []string `json:"tags"`
}

type NotebookResponse struct {
    ID    uuid.UUID `json:"id"`
    Title string    `json:"title"`
    // ...
}
```

**Guidelines:**
- Separate request/response DTOs
- Add validation tags
- Use JSON tags for API contracts
- Don't include business logic

#### `/mappers/`
**Purpose:** Convert between domain entities and DTOs.

```go
func ToNotebookResponse(notebook *entities.Notebook) *dtos.NotebookResponse {
    return &dtos.NotebookResponse{
        ID:    notebook.ID,
        Title: notebook.Title,
        // ...
    }
}
```

**Guidelines:**
- Keep conversion logic simple
- Handle nil values appropriately
- Don't add business logic

#### `/ports/` (optional)
**Purpose:** Define interfaces for external services needed by use cases.

```go
type EmbeddingProvider interface {
    Embed(ctx context.Context, text string) ([]float64, error)
}
```

---

## `/internal/infrastructure` — Infrastructure Layer
**Purpose:** Technical details and external integrations.

### `/config/`
**Purpose:** Configuration loading and management.

```go
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    Gemini   GeminiConfig
}
```

**Guidelines:**
- Support environment variables
- Provide defaults
- Validate at startup

### `/persistence/`
**Purpose:** Repository implementations (adapters).

```go
type PostgresNotebookRepository struct {
    pool *pgxpool.Pool
}

func (r *PostgresNotebookRepository) Save(ctx context.Context, notebook *entities.Notebook) error {
    // Database operations
}
```

**Guidelines:**
- Implement domain repository interfaces
- Handle database-specific concerns
- Return domain errors, not DB errors
- Use connection pooling

### `/cache/`
**Purpose:** Caching implementations (Redis, in-memory, etc.).

### `/external/`
**Purpose:** External API clients (Gemini, OpenAI, etc.).

### `/messaging/`
**Purpose:** Message brokers, event publishers/subscribers.

---

## `/internal/interfaces` — Interface Layer
**Purpose:** Adapters that allow external systems to interact with the application.

### `/http/`
**Purpose:** HTTP API layer.

#### `/handlers/`
**Purpose:** HTTP request handlers (controllers).

```go
type NotebookHandler struct {
    useCase usecases.NotebookUseCase
    logger  *logger.Logger
}

func (h *NotebookHandler) Create(w http.ResponseWriter, r *http.Request) {
    // Parse request, call use case, send response
}
```

**Guidelines:**
- Thin layer - delegate to use cases
- Handle HTTP concerns (status codes, headers)
- Don't include business logic
- Use middleware for cross-cutting concerns

#### `/middleware/`
**Purpose:** HTTP middleware (logging, auth, CORS, recovery).

```go
func Logger(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Log request
        next.ServeHTTP(w, r)
    })
}
```

**Guidelines:**
- Keep middleware focused on one concern
- Use standard middleware signature
- Don't modify request body

#### `/routes/`
**Purpose:** Route registration and URL mapping.

```go
func Setup(router *mux.Router, notebookHandler *handlers.NotebookHandler) {
    api := router.PathPrefix("/api/v1").Subrouter()
    notebooks := api.PathPrefix("/notebooks").Subrouter()
    notebooks.HandleFunc("", notebookHandler.Create).Methods(http.MethodPost)
}
```

**Guidelines:**
- Group related routes
- Use path variables for IDs
- Apply middleware at appropriate levels

### `/grpc/` (future)
**Purpose:** gRPC service implementations.

---

## `/pkg` — Reusable Packages
**Purpose:** Libraries that could be used by other projects.

### `/logger/`
**Purpose:** Structured logging wrapper.

```go
log := logger.New(logger.LogLevel("info"), "json")
log.Info("server started", "port", 8080)
log.Error("failed to connect", "error", err)
```

### `/validator/`
**Purpose:** Input validation utilities.

### `/uuid/`
**Purpose:** UUID generation and parsing.

### `/indexer/pgvector/`
**Purpose:** Vector embeddings storage in PostgreSQL.

```go
indexer, _ := pgvector.NewIndexer(ctx, &pgvector.Config{
    Pool:     pool,
    Dimension: 768,
})
```

### `/retriever/pgvector/`
**Purpose:** Vector similarity search.

### `/constants/`
**Purpose:** Application-wide constants.

### `/errors/`
**Purpose:** Common error types and utilities.

### `/utils/`
**Purpose:** General utility functions.

**Guidelines for `/pkg`:**
- No internal dependencies
- Well-documented public APIs
- Version independently
- Keep focused and small

---

## `/test` — Test Code
**Purpose:** All test files organized by type.

### `/unit/`
**Purpose:** Unit tests for individual functions/methods.

**Guidelines:**
- Test pure functions in isolation
- Use mocks for dependencies
- Fast execution

### `/integration/`
**Purpose:** Tests of multiple components working together.

**Guidelines:**
- Test with real database
- Test repository implementations
- Slower than unit tests

### `/e2e/`
**Purpose:** End-to-end tests of full workflows.

**Guidelines:**
- Test through HTTP interface
- Test complete user scenarios
- Slowest, run in CI

### `/mocks/`
**Purpose:** Mock implementations for testing.

---

## `/scripts` — Build and Dev Scripts
**Purpose:** Automation scripts for development, build, deployment.

```bash
scripts/
├── build.sh       # Build application
├── test.sh        # Run all tests
├── migrate.sh     # Database migrations
└── deploy.sh      # Deployment script
```

---

## `/docs` — Documentation
**Purpose:** Project documentation.

```
docs/
├── api/           # API documentation
├── architecture/  # Architecture diagrams
└── guides/        # User/developer guides
```

---

## `/build` — Build Output
**Purpose:** Compiled binaries and build artifacts.

**Guidelines:**
- Add to `.gitignore`
- Clean before builds

---

## `/bin` — Executable Binaries
**Purpose:** Installed binary tools.

**Guidelines:**
- Add to `.gitignore`
- Contains final executables

---

## Quick Reference: Where to Put Code

| You Want To... | Put It In... |
|----------------|--------------|
| Define a business concept (Notebook, Document) | `/internal/core/domain/entities/` |
| Define data access contract | `/internal/core/domain/repositories/` |
| Implement business workflow | `/internal/core/application/usecases/` |
| Handle HTTP request | `/internal/interfaces/http/handlers/` |
| Store in PostgreSQL | `/internal/infrastructure/persistence/` |
| Call external API | `/internal/infrastructure/external/` |
| Create reusable utility | `/pkg/` |
| Add new CLI command | `/cmd/` |
| Write unit test | `/test/unit/` |

---

## File Naming Conventions

| Pattern | Meaning |
|---------|---------|
| `notebook.go` | Main code for notebook |
| `notebook_test.go` | Tests for notebook |
| `notebook_test.go` | Unit test file |
| `mock_notebook_repo.go` | Mock implementation |
| `doc.go` | Package documentation |
