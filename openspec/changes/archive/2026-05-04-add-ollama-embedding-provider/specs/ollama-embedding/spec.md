## Purpose

Enable Ollama as an embedding provider for text vectorization during document ingestion and retrieval.

## Requirements

### Requirement: Ollama Embedder Initialization
The system SHALL initialize an Ollama embedder when `EMBEDDING_PROVIDER` is set to `ollama`, using the official `eino-ext/components/embedding/ollama` package.

#### Scenario: Create Ollama embedder with valid config
- **WHEN** `EMBEDDING_PROVIDER=ollama`, `EMBEDDING_BASE_URL=http://localhost:11434`, `EMBEDDING_MODEL=embeddinggemma`
- **THEN** the system SHALL create an Ollama embedder using `ollama.NewEmbedder()`
- **AND** the embedder SHALL implement the `embedding.Embedder` interface
- **AND** use the configured base URL, model, and timeout

#### Scenario: Ollama embedder with default base URL
- **WHEN** `EMBEDDING_PROVIDER=ollama` and `EMBEDDING_BASE_URL` is empty
- **THEN** the system SHALL use `http://localhost:11434` as the default base URL

#### Scenario: Ollama service unavailable
- **WHEN** the Ollama service at the configured base URL is not reachable
- **THEN** the system SHALL return an error at initialization time
- **AND** the error SHALL indicate the connection failure

### Requirement: Ollama Embedding Factory Registration
The system SHALL register `ollama` as a valid provider in the embedding factory alongside `gemini` and `llamacpp`.

#### Scenario: Provider constant registered
- **WHEN** the provider registry is initialized
- **THEN** `ProviderOllama` with value `"ollama"` SHALL be available

### Requirement: Ollama Text Embedding
The system SHALL generate text embeddings using Ollama's `/api/embed` endpoint via the `EmbedStrings()` method.

#### Scenario: Generate embeddings for text chunks
- **WHEN** the ingestion pipeline processes document chunks with `EMBEDDING_PROVIDER=ollama`
- **THEN** the Ollama embedder SHALL generate vectors for each chunk via `EmbedStrings()`
- **AND** the vectors SHALL be stored in the knowledge and sentence tables

### Requirement: Ollama Vision Embedding Not Supported
The system SHALL NOT support vision embedding for Ollama.

#### Scenario: Vision embedding requested with Ollama
- **WHEN** `CreateVisionEmbedder()` is called with Ollama provider
- **THEN** the system SHALL return an error indicating vision embedding is not supported
