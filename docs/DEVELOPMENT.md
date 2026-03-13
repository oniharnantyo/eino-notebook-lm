# Development Guide

This guide shows how to develop features following the **Hexagonal Architecture** pattern.

## Hexagonal Architecture Diagram

```
                    ┌─────────────────────────────────────┐
                    │         EXTERNAL WORLD              │
                    │    (Users, Other Systems, APIs)     │
                    └─────────────────────────────────────┘
                                     │
                                     │ requests/responses
                                     ▼
┌───────────────────────────────────────────────────────────────────────┐
│                         ┌─────────────────┐                            │
│                         │  INTERFACE      │                            │
│                         │  ADAPTERS       │                            │
│                         │  (Primary)      │                            │
│                         │  ┌───────────┐  │                            │
│                         │  │   HTTP    │  │  REST API, GraphQL,        │
│   ┌─────────────────┐   │  │  Handlers │  │  WebSockets                │
│   │                 │   │  └───────────┘  │                            │
│   │   SECONDARY     │   │  ┌───────────┐  │                            │
│   │   ADAPTERS      │   │  │    CLI    │  │  Command Line Interface   │
│   │                 │   │  └───────────┘  │                            │
│   │ ┌─────────────┐ │   │  ┌───────────┐  │                            │
│   │ │  Database   │ │   │  │   gRPC    │  │  RPC (optional)            │
│   │ │  (Postgres) │ │   │  └───────────┘  │                            │
│   │ └─────────────┘ │   │                 │                            │
│   │ ┌─────────────┐ │   │                 │                            │
│   │ │    Cache    │ │   │                 │                            │
│   │ │   (Redis)   │ │   │                 │                            │
│   │ └─────────────┘ │   └─────────────────┘                            │
│   │ ┌─────────────┐ │        │                                         │
│   │ │   Message   │ │        │ calls                                   │
│   │ │    Queue    │ │        │                                         │
│   │ └─────────────┘ │        ▼                                         │
│   │                 │   ┌─────────────────┐                            │
│   │                 │   │   APPLICATION   │                            │
│   │                 │   │      LAYER      │                            │
│   │                 │   │  ┌───────────┐  │                            │
│   │                 │   │  │ Use Cases │  │  Business Logic            │
│   │                 │   │  └───────────┘  │  Orchestration             │
│   │                 │   └─────────────────┘                            │
│   │                 │          │                                         │
│   │                 │          │ uses                                    │
│   │                 │          ▼                                         │
│   │                 │   ┌─────────────────┐                            │
│   │                 │   │     DOMAIN      │                            │
│   │                 │   │      LAYER      │                            │
│   │                 │   │  ┌───────────┐  │                            │
│   │                 │   │  │ Entities  │  │  Business Rules            │
│   │                 │   │  └───────────┘  │                            │
│   │                 │   │  ┌───────────┐  │                            │
│   │                 │   │  │  Ports    │  │  Repository Interfaces    │
│   │                 │   │  └───────────┘  │  (Domain Services)         │
│   │                 │   └─────────────────┘                            │
│   │                 │          ▲                                         │
│   │                 │          │ implements                              │
│   │                 │          │                                         │
│   └─────────────────┘       ──┘                                         │
│                                                                              │
│                           THE HEXAGON                                       │
└──────────────────────────────────────────────────────────────────────────────┘
```

## Flow of a Request

```
User Request
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ 1. INTERFACE LAYER (Primary Adapter)                        │
│    • HTTP Handler receives request                          │
│    • Validates request format                               │
│    • Calls Use Case                                         │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. APPLICATION LAYER (Use Case)                             │
│    • Orchestrates business logic                            │
│    • Uses Domain entities                                   │
│    • Calls Repository interface (Port)                      │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. DOMAIN LAYER (Core Business Logic)                       │
│    • Entity validation                                      │
│    • Business rules enforcement                             │
│    • Repository interface definition                        │
└─────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. INFRASTRUCTURE LAYER (Secondary Adapter)                 │
│    • Repository implementation                              │
│    • Database access (PostgreSQL, etc.)                     │
│    • Returns Domain entities                                │
└─────────────────────────────────────────────────────────────┘
```

## Creating a New Feature: Complete Example

Let's create a **"Get Notebook by ID"** feature from scratch.

### Step 1: Define Domain Layer (Port)

First, define the repository interface in the domain layer. This is the "Port" that external adapters will implement.

**File:** `internal/core/domain/repositories/notebook.go`

```go
package repositories

import (
    "context"
    "github.com/oniharnantyo/eino-notebook/pkg/uuid"
)

// NotebookRepository is a PORT - defines what we need from storage
type NotebookRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error)
    Save(ctx context.Context, notebook *entities.Notebook) error
    // ... more methods
}
```

### Step 2: Implement Infrastructure Layer (Adapter)

Create the actual database implementation. This is the "Adapter" that implements the Port.

**File:** `internal/infrastructure/persistence/postgres.go`

```go
package persistence

import (
    "context"
    "github.com/oniharnantyo/eino-notebook/internal/core/domain/entities"
    "github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
)

// PostgresNotebookRepository is an ADAPTER - implements the Port
type PostgresNotebookRepository struct {
    db *pgx.Conn
}

// Ensure it implements the interface
var _ repositories.NotebookRepository = (*PostgresNotebookRepository)(nil)

func NewPostgresNotebookRepository(db *pgx.Conn) repositories.NotebookRepository {
    return &PostgresNotebookRepository{db: db}
}

func (r *PostgresNotebookRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.Notebook, error) {
    // SQL query
    row := r.db.QueryRow(ctx,
        "SELECT id, title, description, content, status, tags, created_at, updated_at FROM notebooks WHERE id = $1",
        id,
    )

    // Scan into entity
    var n entities.Notebook
    if err := row.Scan(&n.ID, &n.Title, &n.Description, &n.Content, &n.Status, &n.Tags, &n.CreatedAt, &n.UpdatedAt); err != nil {
        return nil, err
    }

    return &n, nil
}

func (r *PostgresNotebookRepository) Save(ctx context.Context, notebook *entities.Notebook) error {
    // SQL insert/update
    _, err := r.db.Exec(ctx,
        `INSERT INTO notebooks (id, title, description, content, status, tags, created_at, updated_at)
         VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
         ON CONFLICT (id) DO UPDATE SET title=$2, description=$3, content=$4, updated_at=$8`,
        notebook.ID, notebook.Title, notebook.Description, notebook.Content,
        notebook.Status, notebook.Tags, notebook.CreatedAt, notebook.UpdatedAt,
    )
    return err
}
```

### Step 3: Create Application Layer (Use Case)

The use case orchestrates the business logic using the repository interface.

**File:** `internal/core/application/usecases/notebook.go`

```go
package usecases

import (
    "context"
    "github.com/oniharnantyo/eino-notebook/internal/core/domain/repositories"
    "github.com/oniharnantyo/eino-notebook/internal/core/application/dtos"
    "github.com/oniharnantyo/eino-notebook/internal/core/application/mappers"
)

// NotebookUseCase defines business operations
type NotebookUseCase interface {
    GetByID(ctx context.Context, id string) (*dtos.NotebookResponse, error)
    Create(ctx context.Context, req *dtos.CreateNotebookRequest) (*dtos.NotebookResponse, error)
}

type notebookUseCase struct {
    notebookRepo repositories.NotebookRepository  // Depends on interface, not implementation!
}

func NewNotebookUseCase(repo repositories.NotebookRepository) NotebookUseCase {
    return &notebookUseCase{
        notebookRepo: repo,
    }
}

func (uc *notebookUseCase) GetByID(ctx context.Context, id string) (*dtos.NotebookResponse, error) {
    // Parse ID
    uid, err := mappers.ParseID(id)
    if err != nil {
        return nil, errors.NewValidationError("invalid ID format")
    }

    // Use repository (doesn't know if it's Postgres, MySQL, In-Memory, etc.)
    notebook, err := uc.notebookRepo.FindByID(ctx, uid)
    if err != nil {
        return nil, errors.NewInternalError("failed to fetch notebook", err)
    }
    if notebook == nil {
        return nil, errors.NewNotFoundError("notebook")
    }

    // Return DTO (not the entity directly!)
    return mappers.ToNotebookResponse(notebook), nil
}
```

### Step 4: Create Interface Layer (Handler)

The HTTP handler is the "Primary Adapter" that receives HTTP requests and calls the use case.

**File:** `internal/interfaces/http/handlers/notebook.go`

```go
package handlers

import (
    "net/http"
    "github.com/gorilla/mux"
    "github.com/oniharnantyo/eino-notebook/internal/core/application/usecases"
)

type NotebookHandler struct {
    useCase usecases.NotebookUseCase  // Depends on interface
    logger  *logger.Logger
}

func NewNotebookHandler(uc usecases.NotebookUseCase, log *logger.Logger) *NotebookHandler {
    return &NotebookHandler{
        useCase: uc,
        logger:  log,
    }
}

func (h *NotebookHandler) GetByID(w http.ResponseWriter, r *http.Request) {
    // Extract from HTTP request
    vars := mux.Vars(r)
    id := vars["id"]

    // Call use case
    notebook, err := h.useCase.GetByID(r.Context(), id)
    if err != nil {
        h.logger.Error("failed to get notebook", "id", id, "error", err)
        h.respondWithError(w, http.StatusNotFound, err.Error())
        return
    }

    // Return HTTP response
    h.respondWithJSON(w, http.StatusOK, notebook)
}
```

### Step 5: Wire Dependencies

In your main server startup, wire everything together:

**File:** `cmd/serve.go`

```go
func runServer() error {
    // 1. Infrastructure Layer - Create adapter
    dbConn := connectToDatabase()
    notebookRepo := persistence.NewPostgresNotebookRepository(dbConn)

    // OR use in-memory for testing
    // notebookRepo := persistence.NewInMemoryNotebookRepository()

    // 2. Application Layer - Create use case with repository
    notebookUseCase := usecases.NewNotebookUseCase(notebookRepo)

    // 3. Interface Layer - Create handler with use case
    notebookHandler := handlers.NewNotebookHandler(notebookUseCase, logger)

    // 4. Setup routes
    router := mux.NewRouter()
    router.HandleFunc("/api/v1/notebooks/{id}", notebookHandler.GetByID).Methods("GET")

    // Start server
    return http.Serve(router)
}
```

## Key Principles

### 1. Dependency Inversion

**❌ WRONG - Direct dependency:**
```go
type NotebookUseCase struct {
    repo *PostgresNotebookRepository  // Depends on concrete implementation!
}
```

**✅ RIGHT - Depends on abstraction:**
```go
type NotebookUseCase struct {
    repo repositories.NotebookRepository  // Depends on interface!
}
```

### 2. Entities vs DTOs

**❌ WRONG - Expose entities externally:**
```go
func GetByID(id string) (*entities.Notebook, error) {
    // Returns domain entity directly
}
```

**✅ RIGHT - Use DTOs for external communication:**
```go
func GetByID(id string) (*dtos.NotebookResponse, error) {
    // Returns DTO, entity stays in domain
}
```

### 3. Business Logic Placement

**❌ WRONG - Business logic in handler:**
```go
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    // Validation here?
    if req.Title == "" {
        http.Error(w, "title required", 400)
        return
    }
    // Business logic in handler = bad
}
```

**✅ RIGHT - Business logic in domain/use case:**
```go
// Handler just delegates
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
    notebook, err := h.useCase.Create(r.Context(), &req)
}

// Use case orchestrates
func (uc *UseCase) Create(ctx context.Context, req *dtos.CreateNotebookRequest) {
    notebook := entities.NewNotebook(req.Title, ...)  // Domain validates
    uc.repo.Save(ctx, notebook)
}

// Entity contains business rules
func NewNotebook(title string, ...) (*Notebook, error) {
    if title == "" {
        return nil, errors.ErrEmptyTitle  // Domain error
    }
}
```

## Adding a New Feature Checklist

When adding a new feature, follow this order:

```
1. DOMAIN LAYER
   □ Create/modify entity in internal/core/domain/entities/
   □ Add/update repository interface in internal/core/domain/repositories/
   □ Define domain errors if needed

2. APPLICATION LAYER
   □ Create DTO in internal/core/application/dtos/
   □ Create mapper in internal/core/application/mappers/
   □ Create/update use case in internal/core/application/usecases/

3. INFRASTRUCTURE LAYER
   □ Implement repository in internal/infrastructure/persistence/
   □ Add migration if needed

4. INTERFACE LAYER
   □ Create handler in internal/interfaces/http/handlers/
   □ Add route in internal/interfaces/http/routes/

5. WIRING
   □ Update cmd/serve.go to wire dependencies
   □ Add tests in test/unit/ or test/integration/

6. DOCUMENTATION
   □ Update README.md with API documentation
```

## Testing Strategy

### Unit Tests (Domain Layer)
```go
func TestNotebook_NewNotebook(t *testing.T) {
    // Test entity validation logic
    notebook, err := entities.NewNotebook("", "desc", "content", nil)
    assert.Error(t, err)
    assert.Nil(t, notebook)
}
```

### Integration Tests (Application Layer)
```go
func TestNotebookUseCase_GetByID(t *testing.T) {
    // Use mock repository
    mockRepo := &MockNotebookRepository{}
    useCase := usecases.NewNotebookUseCase(mockRepo)

    result, err := useCase.GetByID(context.Background(), "123")
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### E2E Tests (Interface Layer)
```go
func TestHTTP_GetNotebook(t *testing.T) {
    // Start test server
    server := setupTestServer()
    defer server.Close()

    // Make HTTP request
    resp := request(server, "GET", "/api/v1/notebooks/123")
    assert.Equal(t, 200, resp.StatusCode)
}
```

## Common Patterns

### Repository Pattern

```go
// 1. Define interface in domain
type UserRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error)
}

// 2. Implement in infrastructure
type PostgresUserRepository struct {
    db *pgx.Conn
}

func (r *PostgresUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*entities.User, error) {
    // SQL implementation
}

// 3. Use in application layer
type UserUseCase struct {
    userRepo UserRepository  // Interface, not implementation!
}
```

### Factory Pattern

```go
// Factory for creating entities
func NewNotebook(title, description, content string, tags []string) (*Notebook, error) {
    // Validation
    // Default values
    // Return entity
}
```

### Mapper Pattern

```go
// Entity → DTO
func ToNotebookResponse(n *entities.Notebook) *dtos.NotebookResponse {
    return &dtos.NotebookResponse{
        ID:    n.ID,
        Title: n.Title,
        // ... map fields
    }
}

// DTO → Entity (via factory)
func ToEntity(req *dtos.CreateNotebookRequest) (*entities.Notebook, error) {
    return entities.NewNotebook(req.Title, req.Description, req.Content, req.Tags)
}
```

## Quick Reference

| Layer | Purpose | Depends On | Example |
|-------|---------|------------|---------|
| **Domain** | Business logic, rules | Nothing (pure) | `Notebook`, `NotebookStatus`, `ErrEmptyTitle` |
| **Application** | Use cases, orchestration | Domain | `NotebookUseCase`, `CreateNotebookRequest` |
| **Infrastructure** | External concerns | Domain (implements interfaces) | `PostgresNotebookRepository`, `Config` |
| **Interface** | External communication | Application | `NotebookHandler`, `HTTPMiddleware` |

## FAQs

**Q: Where do I put validation logic?**
- Domain validation: In entities (`notebook.Validate()`)
- Input validation: In handlers (format checks)
- Business validation: In use cases

**Q: Can I call infrastructure from domain?**
- NO! Domain must be pure. Use dependency injection.

**Q: How do I switch from Postgres to MongoDB?**
- Just create `MongoNotebookRepository` implementing `NotebookRepository`
- Change one line in `serve.go` wiring

**Q: Should handlers call repositories directly?**
- NO! Handlers → Use Cases → Repositories
- This keeps business logic reusable across interfaces

**Q: What about shared code?**
- Put in `pkg/` if truly generic
- Put in domain if business-related
- Create a domain service if complex
