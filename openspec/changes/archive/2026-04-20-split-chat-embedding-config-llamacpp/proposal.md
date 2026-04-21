## Why

Current architecture couples chat and embedding configurations with shared API keys and endpoints per provider. This prevents using different services (e.g., Gemini for chat, llama.cpp for embeddings) and limits deployment flexibility. Additionally, the provider inference from model names (`gemini/model`) creates unnecessary coupling.

## What Changes

- **New Config Structure**: Replace `ModelConfig.ChatModel/EmbeddingModel` with independent `ChatConfig` and `EmbeddingConfig` structs
- **Factory Pattern**: Create `chat_factory.go` and `embedding_factory.go` for provider-agnostic model/embedder creation
- **llama.cpp Integration**: Add new `ProviderLlamaCpp` with HTTP-based embedder supporting batch requests and configurable prompt templates
- **Prompt Template System**: Built-in templates (default, with-instruction, vision) with Go template syntax for variable substitution
- **BREAKING**: Environment variables change from `GEMINI_API_KEY`, `CHAT_MODEL` (prefix format) to `CHAT_PROVIDER`, `CHAT_MODEL`, `CHAT_API_KEY`, `EMBEDDING_PROVIDER`, `EMBEDDING_MODEL`, `EMBEDDING_API_KEY`, `EMBEDDING_BASE_URL`, `EMBEDDING_PROMPT_TEMPLATE`

## Capabilities

### New Capabilities
- `llamacpp-embedding`: Text embeddings via llama.cpp HTTP API with configurable prompt templates and independent credentials

### Modified Capabilities
- None (internal refactoring only; external API behavior unchanged)

## Impact

**Affected Files:**
- `internal/infrastructure/config/config.go`: Add `ChatConfig`, `EmbeddingConfig`; remove deprecated `GeminiConfig` fields
- `cmd/serve.go`: Replace provider inference with factory-based initialization; remove shared client logic
- `pkg/model/provider.go`: Add `ProviderLlamaCpp` constant
- `.env.example`: New environment variable structure

**New Files:**
- `pkg/model/chat_factory.go`: Chat model factory
- `pkg/model/embedding_factory.go`: Embedder factory
- `pkg/embedding/llamacpp/embedder.go`: llama.cpp embedder implementation
- `pkg/embedding/llamacpp/config.go`: llama.cpp-specific config
- `pkg/embedding/llamacpp/doc.go`: Documentation
- `pkg/embedding/templates/registry.go`: Template registry
- `pkg/embedding/templates/default.go`: Built-in templates

**Dependencies:**
- No new external dependencies (uses `net/http` for llama.cpp calls)
