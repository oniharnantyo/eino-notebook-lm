# Plan: Async Knowledge Ingestion

**Date**: 2026-03-24
**Status**: Draft

## Context

Implement asynchronous knowledge ingestion via `POST /api/v1/notebooks/{notebookId}/knowledges` with an `async: true` body flag. When async is enabled, the API returns immediately with a tracking status, allowing long-running ingestion (content extraction, chunking, embedding) to proceed in the background.

This follows the existing pattern from streaming responses (`internal/interfaces/http/handlers/response.go`) which uses status states and background goroutines.

## Implementation Approach

### 1. Add Status Tracking to Source Entity

**File**: `internal/core/domain/entities/source.go`

Add status field and helper methods:

```go
type SourceStatus string

const (
    SourceStatusPending    SourceStatus = "pending"
    SourceStatusProcessing SourceStatus = "processing"
    SourceStatusCompleted  SourceStatus = "completed"
    SourceStatusFailed     SourceStatus = "failed"
)

// Add to Source struct:
Status SourceStatus `json:"status" db:"status"`
Error  *string      `json:"error,omitempty" db:"error"`

// Add methods:
func (s *Source) MarkProcessing()
func (s *Source) MarkCompleted()
func (s *Source) MarkFailed(err error)
```

### 2. Database Migration

**New File**: `migrations/000008_add_source_status.up.sql`

```sql
ALTER TABLE sources ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'pending';
ALTER TABLE sources ADD COLUMN IF NOT EXISTS error TEXT;
CREATE INDEX IF NOT EXISTS idx_sources_status ON sources(status) WHERE deleted_at IS NULL;
UPDATE sources SET status = 'completed' WHERE content IS NOT NULL AND chunk_count > 0;
```

### 3. Update Source Repository

**File**: `internal/infrastructure/persistence/source.go`

- Update `Create`, `GetByID`, `scanSource` to include status and error fields
- Add new method:

```go
func (r *PostgresSourceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status entities.SourceStatus, errMsg *string) error
```

### 4. Update DTOs

**File**: `internal/core/application/dtos/source.go`

Add to `SourceResponse`:

```go
Status string  `json:"status"`
Error  *string `json:"error,omitempty"`
```

**File**: `internal/core/application/dtos/knowledge.go`

Add async response DTO:

```go
type AsyncKnowledgeResponse struct {
    SourceID        uuid.UUID `json:"source_id"`
    Status          string    `json:"status"`
    StatusStreamURL string    `json:"status_stream_url"`
}
```

### 5. Update Knowledge Handler

**File**: `internal/interfaces/http/handlers/knowledge.go`

Modify `Create` method (line 57):
- Check for `async` form field
- If async=true: create source with "pending" status, spawn goroutine, return 202 Accepted
- If async=false: existing synchronous behavior

Add new method:

```go
func (h *KnowledgeHandler) processAsync(ctx context.Context, sourceID uuid.UUID, req *AsyncProcessRequest)
```

Add SSE status streaming endpoint:

```go
func (h *KnowledgeHandler) StreamSourceStatus(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    flusher, _ := w.(http.Flusher)

    // Poll database every 500ms until terminal state
    ticker := time.NewTicker(500 * time.Millisecond)
    defer ticker.Stop()

    var lastStatus string
    for {
        select {
        case <-r.Context().Done():
            return // Client disconnected
        case <-ticker.C:
            source, _ := h.sourceUseCase.GetByID(ctx, sourceID)

            // Send event only if status changed
            if source.Status != lastStatus {
                h.sendSSEEvent(w, flusher, source)
                lastStatus = source.Status
            }

            // Close connection on terminal state
            if source.Status == "completed" || source.Status == "failed" {
                return
            }
        }
    }
}
```

### 6. Update Routes

**File**: `internal/interfaces/http/routes/routes.go`

Add route under notebooks:

```go
notebooks.HandleFunc("/{notebookId}/knowledges", knowledgeHandler.Create).Methods(http.MethodPost)
notebooks.HandleFunc("/{notebookId}/knowledges/status/{sourceId}/stream", knowledgeHandler.StreamSourceStatus).Methods(http.MethodGet)
```

## Files to Modify

| File | Change |
|------|--------|
| `internal/core/domain/entities/source.go` | Add Status, Error fields and methods |
| `internal/infrastructure/persistence/source.go` | Add UpdateStatus, update scans |
| `internal/core/application/dtos/source.go` | Add Status, Error to response |
| `internal/core/application/dtos/knowledge.go` | Add AsyncKnowledgeResponse |
| `internal/interfaces/http/handlers/knowledge.go` | Add async handling logic |
| `internal/interfaces/http/routes/routes.go` | Add new routes |
| `migrations/000008_add_source_status.up.sql` | New migration |

## API Design

### Request

```bash
POST /api/v1/notebooks/{notebookId}/knowledges
Content-Type: multipart/form-data

file=@document.pdf
title="My Document"
async=true
```

### Response (202 Accepted when async=true)

```json
{
  "source_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "status_stream_url": "/api/v1/notebooks/{notebookId}/knowledges/status/550e8400-e29b-41d4-a716-446655440000/stream"
}
```

### Status Stream (SSE)

```bash
GET /api/v1/notebooks/{notebookId}/knowledges/status/{sourceId}/stream
Accept: text/event-stream
```

**SSE Events:**

```
event: status
data: {"source_id":"550e8400-...","status":"pending","progress":0}

event: status
data: {"source_id":"550e8400-...","status":"processing","progress":0.3,"message":"Extracting content..."}

event: status
data: {"source_id":"550e8400-...","status":"processing","progress":0.6,"message":"Chunking document..."}

event: status
data: {"source_id":"550e8400-...","status":"processing","progress":0.9,"message":"Generating embeddings..."}

event: status
data: {"source_id":"550e8400-...","status":"completed","progress":1.0}
```

**On Error:**

```
event: status
data: {"source_id":"550e8400-...","status":"failed","error":"Failed to extract content: corrupted PDF"}
```

## Error Handling

- Errors in async goroutine update source status to "failed" with error message
- SSE stream sends final error event before closing connection
- No automatic retry - client can re-submit after fixing issues
- Client can reconnect to SSE stream if connection drops (stream resumes from current status)

## Verification

1. Run migration: `make migrate-up`
2. Test sync mode still works: `curl -F "file=@test.pdf" -F "notebook_id=..." http://localhost:8080/api/v1/notebooks/{id}/knowledges`
3. Test async mode: `curl -F "file=@test.pdf" -F "async=true" -F "notebook_id=..." http://localhost:8080/api/v1/notebooks/{id}/knowledges` - expect 202
4. Test SSE status stream: `curl -N -H "Accept: text/event-stream" http://localhost:8080/api/v1/notebooks/{id}/knowledges/status/{sourceId}/stream`
5. Run tests: `make test`
