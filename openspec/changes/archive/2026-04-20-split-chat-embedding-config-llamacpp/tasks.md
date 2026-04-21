## 1. Config Layer

- [x] 1.1 Add `ChatConfig` struct to `internal/infrastructure/config/config.go`
- [x] 1.2 Add `EmbeddingConfig` struct to `internal/infrastructure/config/config.go`
- [x] 1.3 Update `Config` struct to use `ChatConfig` and `EmbeddingConfig` instead of `ModelConfig`
- [x] 1.4 Add environment variable bindings in `Load()` function (CHAT_*, EMBEDDING_*)
- [x] 1.5 Add default values in `setDefaults()` function for new configs
- [x] 1.6 Update `.env.example` with new environment variable structure
- [x] 1.7 Remove deprecated `ModelConfig` struct (or mark as deprecated for migration period)

## 2. Provider Constants

- [x] 2.1 Add `ProviderLlamaCpp` constant to `pkg/model/provider.go`
- [x] 2.2 Remove `InferProvider()` function from `pkg/model/provider.go` (no longer needed)

## 3. Chat Factory

- [x] 3.1 Create `pkg/model/chat_factory.go` file
- [x] 3.2 Implement `CreateChatModel()` function with provider switch
- [x] 3.3 Implement `createGeminiChatModel()` helper function
- [x] 3.4 Add `createOpenAIChatModel()` helper function (scaffold for future use)

## 4. Embedding Factory

- [x] 4.1 Create `pkg/model/embedding_factory.go` file
- [x] 4.2 Implement `CreateEmbedder()` function with provider switch
- [x] 4.3 Implement `createGeminiEmbedder()` helper function
- [x] 4.4 Implement `createLlamaCppEmbedder()` helper function

## 5. Prompt Template System

- [x] 5.1 Create `pkg/embedding/templates/` directory
- [x] 5.2 Create `pkg/embedding/templates/registry.go` with template registry and `GetTemplate()` function
- [x] 5.3 Create `pkg/embedding/templates/default.go` with `DefaultTemplate`, `WithInstructionTemplate`, `VisionTemplate`
- [x] 5.4 Add template validation logic to ensure Go template syntax is correct

## 6. LlamaCpp Embedder

- [x] 6.1 Create `pkg/embedding/llamacpp/` directory
- [x] 6.2 Create `pkg/embedding/llamacpp/config.go` with `Config` struct (BaseURL, APIKey, Model, Dimension, PromptTemplate, Timeout)
- [x] 6.3 Create `pkg/embedding/llamacpp/doc.go` with package documentation
- [x] 6.4 Create `pkg/embedding/llamacpp/embedder.go` with `LlamaCppEmbedder` struct
- [x] 6.5 Implement `NewEmbedder()` function that validates config and loads template
- [x] 6.6 Implement `EmbedStrings()` function with batch request support
- [x] 6.7 Implement HTTP request building with prompt template rendering
- [x] 6.8 Implement response parsing with index-based extraction and sorting
- [x] 6.9 Add error handling for connection failures, timeouts, and invalid responses

## 7. Application Initialization

- [x] 7.1 Update `cmd/serve.go` to use `model.CreateChatModel()` instead of manual Gemini client creation
- [x] 7.2 Update `cmd/serve.go` to use `model.CreateEmbedder()` instead of manual embedder creation
- [x] 7.3 Remove shared `geminiClient` variable and related logic from `cmd/serve.go`
- [x] 7.4 Remove provider inference code (`InferProvider()` calls) from `cmd/serve.go`
- [x] 7.5 Update logging to show provider and model from new config structures

## 8. Testing

- [x] 8.1 Create `pkg/model/chat_factory_test.go` with unit tests for `CreateChatModel()`
- [x] 8.2 Create `pkg/model/embedding_factory_test.go` with unit tests for `CreateEmbedder()`
- [ ] 8.3 Create `pkg/embedding/llamacpp/embedder_test.go` with unit tests for embedder
- [x] 8.4 Create `pkg/embedding/templates/registry_test.go` with template validation tests
- [ ] 8.5 Add integration test for llama.cpp embedder with mock HTTP server
- [ ] 8.6 Add end-to-end test with config loading and factory initialization

## 9. Documentation

- [x] 9.1 Update `CLAUDE.md` with new configuration structure
- [ ] 9.2 Create migration guide for updating from old env vars to new structure
- [ ] 9.3 Add documentation for llama.cpp provider setup (local vs remote)
- [ ] 9.4 Document prompt template system and how to add custom templates

## 10. Validation

- [ ] 10.1 Run `make build` to ensure no compilation errors
- [ ] 10.2 Run `make test` to ensure all tests pass
- [ ] 10.3 Run `make lint` to ensure code quality
- [ ] 10.4 Test with Gemini chat + Gemini embeddings (existing functionality)
- [ ] 10.5 Test with Gemini chat + llama.cpp embeddings (new functionality)
- [ ] 10.6 Verify error handling for invalid provider names
- [ ] 10.7 Verify error handling for missing required configuration

## 11. Migration

- [ ] 11.1 Update development environment `.env` file with new structure
- [ ] 11.2 Test startup with new configuration
- [ ] 11.3 Verify embeddings work with new config
- [ ] 11.4 Verify chat works with new config
- [ ] 11.5 Remove deprecated code (if doing clean break)
- [ ] 11.6 Tag release (e.g., v1.0.0) to indicate breaking change
