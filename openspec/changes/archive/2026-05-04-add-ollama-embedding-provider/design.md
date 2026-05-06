## Context

The embedding factory in `pkg/model/embedding_factory.go` uses a provider switch pattern: `CreateEmbedder()` maps `EmbeddingConfig.Provider` to the appropriate constructor. Current providers are `gemini` (via eino-ext Gemini SDK) and `llamacpp` (custom HTTP client in `pkg/embedding/llamacpp`).

The `EmbeddingConfig` struct already has `BaseURL`, `Model`, `Dimension`, and `Timeout` fields that map cleanly to Ollama's needs. The official `github.com/cloudwego/eino-ext/components/embedding/ollama` package implements `embedding.Embedder` with `EmbedStrings()` — the same interface used throughout the pipeline.

## Goals / Non-Goals

**Goals:**
- Add Ollama as a third embedding provider option via `EMBEDDING_PROVIDER=ollama`
- Reuse existing configuration fields — no new env vars

**Non-Goals:**
- Vision embedding support for Ollama (Ollama doesn't support multimodal embeddings)
- Changing the embedding pipeline or StorageStage
- Adding Ollama chat model support (out of scope)

## Decisions

### 1. Use official eino-ext Ollama package

**Decision:** Use `github.com/cloudwego/eino-ext/components/embedding/ollama` directly, no custom HTTP client.

**Rationale:** The package is already in the eino-ext ecosystem, implements the `Embedder` interface, and handles the Ollama `/api/embed` API. No reason to reimplement.

**Alternative considered:** Custom HTTP client like LlamaCpp. Rejected — Ollama has an official eino-ext integration while LlamaCpp predates it.

### 2. Map existing config fields

**Decision:** Map `EmbeddingConfig.BaseURL` → `ollama.EmbeddingConfig.BaseURL`, `Model` → `Model`, `Timeout` → `Timeout`. Ignore `Dimension`, `APIKey`, `PromptTemplate` (Ollama doesn't use these).

**Rationale:** Minimizes config changes. Users already set `EMBEDDING_BASE_URL`, `EMBEDDING_MODEL`, `EMBEDDING_TIMEOUT`. Ollama's default base URL is `http://localhost:11434`.

### 3. No vision embedder for Ollama

**Decision:** `CreateVisionEmbedder()` returns an error for Ollama, same as Gemini.

**Rationale:** Ollama's embedding API is text-only. Vision embedding continues to require LlamaCpp.

## Risks / Trade-offs

**[Risk] Ollama service not running** → Mitigation: Factory returns error at startup, same as other providers. User gets clear message.

**[Trade-off] Dimension validation skipped** → Ollama returns whatever dimension the model produces. The `Dimension` config field is ignored. Acceptable — mismatch would be caught at query time.
