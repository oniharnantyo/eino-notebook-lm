## 1. Domain Layer

- [ ] 1.1 Create `internal/core/domain/entities/artifact.go` — Artifact entity with ArtifactType enum (mindmap, podcast, slides), ArtifactStatus enum (pending, processing, completed, failed), factory function `NewArtifact()`, and state transition methods (MarkProcessing, MarkCompleted, MarkFailed)
- [ ] 1.2 Create `internal/core/domain/repositories/artifact.go` — ArtifactRepository interface with Create, GetByID, ListByNotebook (with type filter + pagination), Update methods; ArtifactFilter struct

## 2. Database Migration

- [ ] 1.3 Create migration `migrations/NNNNNN_create_artifacts_table.up.sql` — artifacts table (id UUID PK, source_id UUID FK, notebook_id UUID FK, type VARCHAR, status VARCHAR, result JSONB, error TEXT, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ) with indexes on notebook_id and type
- [ ] 1.4 Create corresponding down migration

## 3. Infrastructure Layer

- [ ] 1.5 Create `internal/infrastructure/persistence/artifact.go` — PostgresArtifactRepository implementing ArtifactRepository interface, following existing pgxpool patterns from source.go

## 4. Application Layer — DTOs & Mappers

- [ ] 2.1 Create `internal/core/application/dtos/artifact.go` — DTOs: CreateArtifactRequest, ArtifactResponse, ListArtifactsRequest, ListArtifactsResponse, TriggerMindmapResponse, ErrorResponse
- [ ] 2.2 Create `internal/core/application/mappers/artifact.go` — Entity ↔ DTO mapping functions (ToArtifactResponse, ToArtifactListResponse)

## 5. Application Layer — Mindmap Use Case

- [ ] 2.3 Create `internal/core/application/usecases/mindmap/usecase.go` — MindmapUseCase interface with Generate method; mindmapUseCase struct with dependencies (sourceRepo, artifactRepo, chatModel, logger); constructor NewMindmapUseCase
- [ ] 2.4 Implement async generation logic — Generate method creates artifact (pending), spawns goroutine with context.WithoutCancel, defer recover, sends Source.Content to chatModel with mindmap prompt, parses JSON response, updates artifact to completed/failed
- [ ] 2.5 Define the mindmap prompt — System + user prompt template that instructs the LLM to return a hierarchical tree JSON with id, label, summary, children fields

## 6. Application Layer — Artifact CRUD Use Case

- [ ] 2.6 Create `internal/core/application/usecases/artifact/usecase.go` — ArtifactUseCase interface with GetByID, List methods; delegates to ArtifactRepository

## 7. Interface Layer — Handlers & Routes

- [ ] 3.1 Create `internal/interfaces/http/handlers/artifact.go` — ArtifactHandler with GenerateMindmap, GetByID, List methods, following existing handler patterns (mux.Vars, JSON response helpers)
- [ ] 3.2 Update `internal/interfaces/http/routes/routes.go` — Register new routes: POST /{notebookId}/sources/{sourceId}/mindmap, GET /{notebookId}/artifacts, GET /{notebookId}/artifacts/{id}
- [ ] 3.3 Update `cmd/serve.go` — Wire new dependencies: create PostgresArtifactRepository, MindmapUseCase, ArtifactUseCase, ArtifactHandler; pass handler to routes.Setup

## 8. Verification

- [ ] 4.1 Run `make build` to verify compilation
- [ ] 4.2 Run `make lint` to verify code quality
