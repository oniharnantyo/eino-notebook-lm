# Hybrid Search Design for pgvector Retriever

**Date:** 2026-03-21
**Status:** Approved
**Author:** Claude Code

## Overview

Enhance the pgvector retriever to use hybrid search combining BM25 (keyword/lexical) and vector (semantic) search using Reciprocal Rank Fusion (RRF).

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Fusion Strategy | RRF (Rank Fusion) | Robust to score scale differences, no weight tuning needed |
| BM25 Source | timescale/pg_textsearch | Native PostgreSQL extension, true BM25 scoring |
| Execution | Parallel (goroutines) | Better performance, both searches run concurrently |
| RRF Parameters | Configurable | Allows tuning k constant and candidate counts |
| Activation | Force hybrid | Always use both BM25 + vector search |
| Reranker | Extensible | Interface for future implementation, RRF only initially |

## Architecture

```
                    ┌─────────────────────────────────────────┐
                    │          HybridRetriever.Retrieve()      │
                    └─────────────────┬───────────────────────┘
                                      │
                    ┌─────────────────┴───────────────────────┐
                    │         (Parallel via errgroup)          │
                    │                                       │
            ┌───────▼────────┐                    ┌─────────▼────────┐
            │  BM25 Search   │                    │  Vector Search   │
            │  (pg_textsearch│                    │  (pgvector)      │
            │   <@> operator │                    │   <=> operator   │
            └───────┬────────┘                    └─────────┬────────┘
                    │                                       │
            ┌───────▼────────┐                    ┌─────────▼────────┐
            │  Ranked List   │                    │  Ranked List     │
            │  [doc_id, rank]│                    │  [doc_id, rank]  │
            └───────┬────────┘                    └─────────┬────────┘
                    │                                       │
                    └─────────────────┬───────────────────────┘
                                      │
                              ┌───────▼────────┐
                              │  RRF Merger    │
                              │  score = Σ     │
                              │  1/(k + rank)  │
                              └───────┬────────┘
                                      │
                              ┌───────▼────────┐
                              │  [Optional]    │
                              │   Reranker     │
                              └───────┬────────┘
                                      │
                              ┌───────▼────────┐
                              │  Final Results │
                              │  (topK, filter)│
                              └────────────────┘
```

## Database Schema

### Prerequisites

1. PostgreSQL 17 or 18
2. pgvector extension (already installed)
3. pg_textsearch extension

### Installation

```sql
-- Add to postgresql.conf and restart
shared_preload_libraries = 'pg_textsearch'

-- Enable extension
CREATE EXTENSION pg_textsearch;

-- Create BM25 index on content column
CREATE INDEX idx_knowledges_bm25 ON knowledges
USING bm25(content) WITH (text_config='english');
```

### No Schema Changes Required

pg_textsearch creates its own BM25 index directly on the text column. No tsvector column, triggers, or GIN indexes needed.

## RRF Algorithm

### Formula

```
RRF_score(d) = Σ (1 / (k + rank(d)))
```

For each document, sum the reciprocal of its rank (plus k) from each search method.

### Example (k=60)

| Doc ID | Vector Rank | BM25 Rank | RRF Score |
|--------|-------------|-----------|-----------|
| doc_A  | 1           | 3         | 1/61 + 1/63 = 0.0323 |
| doc_B  | 2           | 1         | 1/62 + 1/61 = 0.0325 |
| doc_C  | 3           | 2         | 1/63 + 1/62 = 0.0320 |

**Final Order:** doc_B > doc_A > doc_C

### K Parameter

- **k=60** (default): Standard value, good balance
- **Lower k (e.g., 10)**: Top ranks dominate more
- **Higher k (e.g., 100)**: More balanced across ranks

## Component Design

### Config Changes

```go
type Config struct {
    // ... existing fields ...

    // BM25IndexName is the name of the pg_textsearch BM25 index.
    BM25IndexName string

    // BM25TextConfig is the text search configuration.
    // Default: "english"
    BM25TextConfig string

    // Reranker is an optional reranker for post-processing results.
    // If nil, only RRF fusion is used.
    Reranker Reranker
}

// Reranker interface for future implementation
type Reranker interface {
    Rerank(ctx context.Context, query string, docs []*schema.Document) ([]*schema.Document, error)
}
```

### RetrieveOptions Changes

```go
type RetrieveOptions struct {
    // ... existing fields ...

    // RRFK is the RRF constant (default: 60)
    RRFK int

    // BM25Candidates is the number of candidates to fetch from BM25 (default: 50)
    BM25Candidates int

    // VectorCandidates is the number of candidates to fetch from vector search (default: 50)
    VectorCandidates int

    // SkipRerank bypasses the reranker if one is configured
    SkipRerank bool
}
```

### New Option Functions

```go
func WithRRFK(k int) retriever.Option
func WithBM25Candidates(n int) retriever.Option
func WithVectorCandidates(n int) retriever.Option
func WithSkipRerank(skip bool) retriever.Option
```

## SQL Queries

### BM25 Query

```sql
SELECT
    knowledge_id,
    content,
    metadata,
    content <@> $1 AS bm25_score
FROM knowledges
WHERE content <@> to_bm25query($1, 'idx_knowledges_bm25') < 0
ORDER BY content <@> $1
LIMIT $2
```

Note: `< 0` filter because pg_textsearch returns negative scores (lower = better).

### Vector Query (unchanged)

```sql
SELECT
    knowledge_id,
    content,
    metadata,
    embedding <=> $1 AS distance
FROM knowledges
ORDER BY embedding <=> $1
LIMIT $2
```

## Error Handling

- Use `errgroup.WithContext` for parallel execution
- If either search fails, cancel the other via context
- Return wrapped errors with context about which search failed

## File Structure

```
pkg/retriever/pgvector/
├── config.go          # Modify: Add BM25 fields
├── options.go         # Modify: Add RRF params
├── retriever.go       # Modify: Refactor for hybrid
├── bm25.go            # Create: BM25 query execution
├── rrf.go             # Create: RRF merge algorithm
├── doc.go             # Update: Package docs
└── retriever_test.go  # Create: Unit tests
```

## Implementation Order

1. Add config fields and defaults
2. Add retrieve options and option functions
3. Create `rrf.go` with RRF merge algorithm
4. Create `bm25.go` with BM25 query execution
5. Refactor `retriever.go` for parallel hybrid search
6. Add unit tests
7. Update documentation

## Testing Strategy

1. **Unit tests**: RRF algorithm with known inputs/outputs
2. **Integration tests**: Hybrid search with real PostgreSQL
3. **Benchmarks**: Compare pure vector vs hybrid performance

## Future Enhancements

1. **Reranker implementation**: Cross-encoder model or API-based reranking
2. **Adaptive candidates**: Dynamically adjust candidate counts based on query
3. **Score normalization**: Optional min-max normalization before RRF
4. **Filter pushdown**: Optimize WHERE clause handling for both searches