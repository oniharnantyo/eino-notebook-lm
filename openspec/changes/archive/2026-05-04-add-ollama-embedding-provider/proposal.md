## Why

The project supports Gemini and LlamaCpp embedding providers but not Ollama, which is the most common local LLM runtime. Users running Ollama locally (e.g., with `embeddinggemma` or `nomic-embed-text`) cannot use it for text vectorization without a custom LlamaCpp-compatible setup. The official `eino-ext/components/embedding/ollama` package already implements the `Embedder` interface, making this a straightforward factory integration.

## What Changes

- Add `ProviderOllama` constant to the provider registry
- Add Ollama embedder case in `CreateEmbedder()` factory, using `eino-ext/components/embedding/ollama`
- Reuse existing `EmbeddingConfig` fields (`base_url`, `model`, `timeout`) — no new config fields needed

## Capabilities

### New Capabilities

- `ollama-embedding`: Ollama embedding provider integration using the official eino-ext implementation

### Modified Capabilities

None — no existing spec requirements change.

## Impact

**Modified files:**
- `pkg/model/provider.go` — add `ProviderOllama` constant
- `pkg/model/embedding_factory.go` — add Ollama case and helper function

**New dependency:**
- `github.com/cloudwego/eino-ext/components/embedding/ollama` (already downloaded)

**Configuration:** Set `EMBEDDING_PROVIDER=ollama`, `EMBEDDING_BASE_URL=http://localhost:11434`, `EMBEDDING_MODEL=embeddinggemma`
