## 1. Register Ollama provider

- [ ] 1.1 Add `ProviderOllama Provider = "ollama"` constant in `pkg/model/provider.go`
- [ ] 1.2 Add import `ollamaembedder "github.com/cloudwego/eino-ext/components/embedding/ollama"` in `pkg/model/embedding_factory.go`

## 2. Add factory integration

- [ ] 2.1 Add `case ProviderOllama:` in `CreateEmbedder()` switch calling `createOllamaEmbedder(ctx, cfg)`
- [ ] 2.2 Implement `createOllamaEmbedder()` that maps `EmbeddingConfig` → `ollama.EmbeddingConfig{BaseURL, Model, Timeout}` and calls `ollamaembedder.NewEmbedder()`
- [ ] 2.3 Add `case ProviderOllama:` in `CreateVisionEmbedder()` returning error (Ollama doesn't support vision embedding)

## 3. Verification

- [ ] 3.1 Run `make build` — must compile
- [ ] 3.2 Verify existing tests pass with `make test`
