## 1. Infrastructure & Configuration

 - [x] 1.1 Create `docker-compose.yml` with MinIO service (ports 9000/9001, default credentials, named volume for persistence)
 - [x] 1.2 Add `S3Config` struct to `internal/infrastructure/config/config.go` with fields: Endpoint, AccessKey, SecretKey, Bucket; bind env vars (`S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`); set defaults for MinIO
 - [x] 1.3 Add S3 config validation to `Config.Validate()`
 - [x] 1.4 Add S3 env vars to `.env.example`

## 2. Database Migration

 - [x] 2.1 Create migration `000010_refactor_knowledge_ingestion.up.sql`: DROP `knowledges` table, CREATE `knowledges` (relational: id UUID PK, source_id FK, content TEXT, chunk_index INT, heading_context JSONB, first_page INT, last_page INT, metadata JSONB, created_at TIMESTAMPTZ), CREATE `sentences` (id UUID PK, knowledge_id FK, content TEXT, embedding vector(N), position INT, metadata JSONB, created_at TIMESTAMPTZ) with HNSW index, CREATE `images` (id UUID PK, source_id FK, s3_key TEXT, format TEXT, width INT, height INT, ocr_text TEXT, page_number INT, embedding vector(N), metadata JSONB, created_at TIMESTAMPTZ) with HNSW index
 - [x] 2.2 Create corresponding `000010_refactor_knowledge_ingestion.down.sql` reverse migration

## 3. Domain Entities

- [x] 3.1 Refactor `internal/core/domain/entities/knowledge.go`: remove `SubIndexes`, `KnowledgeSource` type, `SourceType`; add `ChunkIndex int`, `HeadingContext map[string]any`, `FirstPage int`, `LastPage int`; update `NewKnowledge` constructor
- [x] 3.2 Create `internal/core/domain/entities/sentence.go`: Sentence entity with ID, KnowledgeID, Content, Embedding (not stored in entity), Position, Metadata, CreatedAt; NewSentence constructor
- [x] 3.3 Create `internal/core/domain/entities/image.go`: Image entity with ID, SourceID, S3Key, Format, Width, Height, OCRText, PageNumber, Embedding (not stored in entity), Metadata, CreatedAt; NewImage constructor

## 4. Domain Repositories

- [x] 4.1 Refactor `internal/core/domain/repositories/knowledge.go`: update KnowledgeRepository for new schema (Save, FindByID, GetBySourceID, DeleteBySourceID, CountBySourceID); remove vector-related methods
- [x] 4.2 Create `internal/core/domain/repositories/sentence.go`: SentenceRepository interface (Save, SaveBatch, FindByKnowledgeID, DeleteByKnowledgeID, DeleteBySourceID)
- [x] 4.3 Create `internal/core/domain/repositories/image.go`: ImageRepository interface (Save, FindBySourceID, DeleteBySourceID, CountBySourceID)

## 5. Kreuzberg Parser Expansion

- [x] 5.1 Update `pkg/parser/kreuzberg/kreuzberg.go`: expand `KreuzbergExtractResponse` with typed fields — `Chunks []KreuzbergChunk`, `Images []KreuzbergImage`, `Elements []KreuzbergElement`, `Tables []any`, `QualityScore float64`, `Pages []KreuzbergPage`, `URIs []KreuzbergURI`, `Annotations []any`
- [x] 5.2 Create typed structs for Kreuzberg sub-types: `KreuzbergChunk` (Content, ChunkType, Metadata with ChunkIndex, TotalChunks, ByteStart, ByteEnd, FirstPage, LastPage, HeadingContext), `KreuzbergImage` (Data []byte, Format, Width, Height, PageNumber, OCRResult), `KreuzbergElement` (ElementID, ElementType, Text, Metadata)

## 6. S3 Storage Infrastructure

 - [x] 6.1 Create `internal/infrastructure/storage/s3.go`: S3Storage struct with MinIO/S3 client, NewS3Storage constructor, Upload(ctx, key string, data []byte) error, Delete(ctx, key string) error, EnsureBucket(ctx) error; use `minio-go` SDK

## 7. Persistence Layer

- [x] 7.1 Refactor `internal/infrastructure/persistence/knowledge.go`: update to new relational schema (no embedding column), batch insert support for knowledge chunks from Kreuzberg
- [x] 7.2 Create `internal/infrastructure/persistence/sentence.go`: implement SentenceRepository using existing pgvector indexer pattern — SaveBatch with embedding generation, FindByKnowledgeID, DeleteByKnowledgeID with cascade
- [x] 7.3 Create `internal/infrastructure/persistence/image.go`: implement ImageRepository using pgvector indexer — Save with embedding from OCR text, FindBySourceID, DeleteBySourceID with S3 cleanup

## 8. Application Use Cases

- [x] 8.1 Refactor `internal/core/application/usecases/knowledge/usecase.go`: accept pre-chunked Kreuzberg chunks instead of raw content + transformer; save knowledge records to relational DB only (no embedding); return created knowledge IDs
 - [x] 8.2 Create `internal/core/application/usecases/sentence/usecase.go`: SentenceUseCase — take knowledge chunks, split each into sentences via recursive transformer, batch embed and store in pgvector
 - [x] 8.3 Create `internal/core/application/usecases/image/usecase.go`: ImageUseCase — take Kreuzberg images, upload binary to S3, embed OCR text, store metadata + embedding in pgvector
 - [x] 8.4 Refactor `internal/core/application/usecases/source/usecase.go`: restructure `processAsync`/`processSync` for three-level flow — extract via Kreuzberg → store source → save knowledge chunks → split & embed sentences → extract & embed images (sentences and images can run in parallel)
 - [x] 8.5 Update `internal/core/application/usecases/extractor/file_extractor.go`: return structured extraction result (content + chunks + images + metadata) instead of flat `[]*schema.Document`

## 9. DTOs & Mappers

- [x] 9.1 Update `internal/core/application/dtos/knowledge.go`: align CreateKnowledgeRequest/Response with new entity fields (chunk_index, heading_context, page_range); remove source_type, sub_indexes
- [x] 9.2 Create `internal/core/application/dtos/sentence.go`: SentenceResponse DTO
- [x] 9.3 Create `internal/core/application/dtos/image.go`: ImageResponse DTO
- [x] 9.4 Update mappers in `internal/core/application/mappers/` for new entity ↔ DTO conversions

## 10. Wiring & DI

 - [x] 10.1 Update `cmd/serve.go`: initialize S3Storage, SentenceRepository, ImageRepository, SentenceUseCase, ImageUseCase; wire into SourceUseCase; create two separate pgvector indexer instances (sentences, images)
 - [x] 10.2 Update recursive transformer config default: change sentence-level chunk_size (~200) for the new sentence splitting use case

## 11. Verification

 - [x] 11.1 Verify `make build` passes with all changes
 - [x] 11.2 Verify `make lint` passes with no new warnings
 - [x] 11.3 Manual test: upload a PDF, verify source → knowledge chunks → sentences → images are created in DB and S3
