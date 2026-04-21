## Context

The application is a RAG notebook system that ingests sources (PDFs, websites, text), creates vector-embedded knowledge chunks, and supports chat-based Q&A. Users need a way to visually understand source structure without reading the full content. Currently, the only exploration modes are raw text viewing or conversational RAG.

The existing codebase follows Clean Architecture with strict layering: domain entities → repository interfaces → use cases → HTTP handlers. Async processing is already established (knowledge ingestion runs async with status tracking via SSE). LLM calls use the Eino framework's `model.BaseChatModel` abstraction with Gemini as the primary provider.

## Goals / Non-Goals

**Goals:**
- Add a generic `artifacts` table and domain concept that can store LLM-generated outputs of different types (mindmap, future: podcast, slides, etc.)
- Implement async mindmap generation that sends `Source.Content` to Gemini and returns a hierarchical tree JSON
- Provide CRUD API for artifacts (create trigger, list, get by ID)
- Follow existing patterns: status tracking (pending → processing → completed/failed), goroutine-based async processing, constructor injection

**Non-Goals:**
- Frontend rendering of mindmaps (API returns structured JSON only)
- Persisting/regenerating Kreuzberg parser output (use existing `Source.Content`)
- Streaming mindmap generation (sync LLM call, async wrapper)
- Artifact versioning or editing
- Multi-model artifact generation (e.g., mindmap from multiple sources)

## Decisions

### 1. Generic artifacts table vs. separate tables per type

**Decision**: Single `artifacts` table with `type` column and JSONB `result` field.

**Rationale**: The user explicitly wants one table for all generated outputs. Adding a new artifact type (podcast, slides) requires only a new enum value and use case — no schema migration. The JSONB `result` field accommodates type-specific output shapes without ALTER TABLE.

**Alternative**: Separate tables per type gives stronger typing but requires migration for each new type. Overkill given the entity shapes are simple and the type discriminant is known at query time.

### 2. Source.Content as LLM input (no summarization)

**Decision**: Send `Source.Content` directly to Gemini. No summarization or chunking pre-processing.

**Rationale**: Gemini 2.0 supports up to 1M input tokens. Most ingested sources (even 100+ page PDFs) fit within ~40K tokens — well within limits. Adding a summarization step would increase latency and cost for no benefit in the typical case. If a future source exceeds limits, we can truncate and add a warning in the artifact error field.

### 3. Goroutine-based async processing (same pattern as knowledge ingestion)

**Decision**: Use `go func()` with `context.WithoutCancel()` — same pattern as `source.go:processAsync`.

**Rationale**: Consistent with the existing async knowledge ingestion flow. No need to introduce a job queue or message broker for a single-generation task. The artifact's `status` field provides tracking.

**Alternative**: A proper job queue (Redis, Temporal) would be more robust for retries and scaling, but is overkill for the current scope.

### 4. LLM prompt design — structured output

**Decision**: Use a system prompt that instructs Gemini to return a JSON tree with `id`, `label`, `summary`, `children` fields. Parse the JSON response directly.

**Rationale**: Gemini supports structured JSON output. A well-crafted prompt avoids needing a separate parsing/formatting step. The tree shape is simple enough that JSON schema enforcement via prompt is sufficient.

### 5. Artifact entity design

**Decision**: Follow the existing entity pattern (e.g., `Source`) with factory function, status enum, and state transition methods.

```
Artifact struct:
  - ID          uuid.UUID
  - SourceID    uuid.UUID
  - NotebookID uuid.UUID
  - Type        ArtifactType    (enum: mindmap, podcast, slides)
  - Status      ArtifactStatus  (enum: pending, processing, completed, failed)
  - Result      JSONB           (type-specific output)
  - Error       *string
  - CreatedAt   time.Time
  - UpdatedAt   time.Time
```

### 6. Route design

**Decision**: Nest artifact routes under notebooks, with a separate trigger endpoint for mindmap generation.

```
POST   /api/v1/notebooks/{nbId}/sources/{srcId}/mindmap     → trigger generation
GET    /api/v1/notebooks/{nbId}/artifacts                    → list all artifacts
GET    /api/v1/notebooks/{nbId}/artifacts/{id}               → get single artifact
```

The trigger endpoint is specific to mindmap (POST to `/mindmap`), while the read endpoints are generic (`/artifacts`). This keeps the trigger semantic clear while the storage/query layer is unified.

## Risks / Trade-offs

- **[Large sources may exceed LLM context]** → Mitigation: Check `Source.Content` length before LLM call. If too large, set artifact status to `failed` with a descriptive error. Future: add optional summarization.
- **[LLM returns malformed JSON]** → Mitigation: Validate JSON structure after LLM response. On parse failure, mark artifact as `failed` with error details. User can retry.
- **[Generic table may become a dumping ground]** → Mitigation: Define clear `ArtifactType` enum. Each type must have a dedicated use case that validates its specific result shape.
- **[Goroutine failures are silent if status update fails]** → Mitigation: Add `defer recover()` for panics (same as existing `processAsync`). Log all failures via structured logger.