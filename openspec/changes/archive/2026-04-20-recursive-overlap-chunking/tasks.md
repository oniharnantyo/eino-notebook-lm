## 1. Add Dependency & Remove Old Code

- [ ] 1.1 Add `github.com/cloudwego/eino-ext/components/document/transformer/splitter/recursive` dependency via `go get`
- [ ] 1.2 Remove `pkg/transformer/element/` package (transformer.go, config.go, element_type.go, tests)
- [ ] 1.3 Remove `pkg/transformer/` directory if empty after element removal

## 2. Update Configuration

- [ ] 2.1 Replace `ElementTransformerConfig` with `RecursiveSplitterConfig` in `internal/infrastructure/config/config.go` (fields: `ChunkSize`, `OverlapSize`)
- [ ] 2.2 Replace `TRANSFORMER_ELEMENT_*` env vars with `TRANSFORMER_CHUNK_SIZE` (default 4000) and `TRANSFORMER_OVERLAP_SIZE` (default 800)
- [ ] 2.3 Remove element-specific config helpers (`GetIncludedElementTypes`, `ToElementConfig`)
- [ ] 2.4 Add validation: overlap_size MUST be less than chunk_size

## 3. Update Wiring

- [ ] 3.1 Replace the element/markdown type switch in `cmd/serve.go` with `recursive.NewSplitter` initialization using config values
- [ ] 3.2 Remove `element` and `markdown` splitter imports from `cmd/serve.go`
- [ ] 3.3 Update log messages to reflect new transformer type and config values

## 4. Wire Transformer Into Pipeline

- [ ] 4.1 Add document concatenation logic in `knowledgeUseCase.Create()` — merge all page-level docs into one `schema.Document` before splitting
- [ ] 4.2 Call `transformer.Transform()` on the concatenated document to produce chunks
- [ ] 4.3 Enrich each chunk with parent metadata (reference_id, title, source_type, sub_indexes, created_at)
- [ ] 4.4 Pass the transformed chunks to `indexer.Store()` instead of raw documents

## 5. Update .env.example

- [ ] 5.1 Replace `TRANSFORMER_TYPE`, `TRANSFORMER_ELEMENT_INCLUDED_TYPES`, `TRANSFORMER_ELEMENT_MAX_CHUNK_SIZE` with `TRANSFORMER_CHUNK_SIZE` and `TRANSFORMER_OVERLAP_SIZE`

## 6. Verify

- [ ] 6.1 Run `make build` to confirm compilation
- [ ] 6.2 Run `make lint` to check for unused imports/variables
