## Tasks

- [ ] **Task 1: Add Ollama chat model dependency** — Add `github.com/cloudwego/eino-ext/components/model/ollama` to `go.mod`, run `go mod tidy`. Files: `go.mod`, `go.sum`
- [ ] **Task 2: Add Timeout and KeepAlive to ChatConfig** — Add `Timeout time.Duration` and `KeepAlive time.Duration` fields to `ChatConfig` in `config.go` with env bindings for `CHAT_TIMEOUT` and `CHAT_KEEP_ALIVE`. Files: `internal/infrastructure/config/config.go`
- [ ] **Task 3: Add Ollama case to chat factory** — Import ollama model package, add `case ProviderOllama:` to `CreateToolCallingChatModel` switch, implement `createOllamaChatModel` function. Files: `pkg/model/chat_factory.go`
- [ ] **Task 4: Add env vars to .env template** — Add `CHAT_TIMEOUT` and `CHAT_KEEP_ALIVE` as commented-out entries. Files: `.env`
