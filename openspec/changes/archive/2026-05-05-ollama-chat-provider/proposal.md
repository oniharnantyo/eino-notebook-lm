## Why

Ollama is already supported for embeddings but not for chat. Users running local LLMs through Ollama want end-to-end local inference — both embedding and chat — without requiring a cloud API key. This is especially useful for offline development, data privacy, and cost savings with capable models like Qwen3 and Llama3 that support tool calling.

## What Changes

- Add Ollama as a chat provider in `CreateToolCallingChatModel` factory
- Add `Timeout` and `KeepAlive` fields to `ChatConfig` for configurable request timeout and model memory retention
- Add `eino-ext/components/model/ollama` dependency

## Capabilities

### New Capabilities
- `ollama-chat`: Ollama chat model provider supporting Generate, Stream, and tool calling via `ToolCallingChatModel` interface

### Modified Capabilities
- `agent-retrieval`: Chat provider selection now includes Ollama, which affects which models can power the retrieval agent

## Impact

- `internal/infrastructure/config/config.go` — Add `Timeout` and `KeepAlive` to `ChatConfig`
- `pkg/model/chat_factory.go` — Add `ProviderOllama` case and `createOllamaChatModel` function
- `go.mod` — Add `github.com/cloudwego/eino-ext/components/model/ollama` dependency
- `.env` — Add `CHAT_TIMEOUT` and `CHAT_KEEP_ALIVE` environment variables
