# Purpose

Define specifications for llama.cpp embedding provider integration in the Eino notebook application. This capability enables the system to use llama.cpp as an embedding provider for vector storage and retrieval operations.

# Requirements

### Requirement: LlamaCpp embedder configuration
The system SHALL support configuring llama.cpp as an embedding provider via independent configuration parameters.

#### Scenario: Configure llama.cpp embedding provider
- **WHEN** administrator sets `EMBEDDING_PROVIDER=llamacpp`
- **AND** sets `EMBEDDING_MODEL=nomic-embed-text-v1.5`
- **AND** sets `EMBEDDING_DIMENSION=768`
- **AND** sets `EMBEDDING_BASE_URL=http://localhost:8080`
- **THEN** system initializes llama.cpp embedder with specified configuration
- **AND** embedder is ready to generate embeddings

#### Scenario: Optional API key for llama.cpp
- **WHEN** administrator sets `EMBEDDING_PROVIDER=llamacpp`
- **AND** omits `EMBEDDING_API_KEY` or sets it to empty string
- **THEN** system initializes llama.cpp embedder without Authorization header
- **AND** embedder successfully connects to local llama.cpp instance

#### Scenario: API key provided for remote llama.cpp
- **WHEN** administrator sets `EMBEDDING_PROVIDER=llamacpp`
- **AND** sets `EMBEDDING_API_KEY=secret-key`
- **AND** sets `EMBEDDING_BASE_URL=https://remote-llamacpp.example.com`
- **THEN** system includes `Authorization: Bearer secret-key` header in all requests
- **AND** embedder successfully authenticates with remote llama.cpp instance

### Requirement: Prompt template selection
The system SHALL support configurable prompt templates for llama.cpp embedding requests.

#### Scenario: Default prompt template
- **WHEN** administrator sets `EMBEDDING_PROMPT_TEMPLATE=default`
- **THEN** system uses default template with `{{.Text}}` variable substitution
- **AND** request payload contains `{"content": [{"prompt_string": "<text>", "multimodal_data": []}]}`

#### Scenario: Custom prompt template with instruction
- **WHEN** administrator sets `EMBEDDING_PROMPT_TEMPLATE=with-instruction`
- **THEN** system uses template with instruction prefix
- **AND** request payload contains `{"content": [{"prompt_string": "Embed this text: <text>", "multimodal_data": []}]}`

#### Scenario: Invalid template name falls back to default
- **WHEN** administrator sets `EMBEDDING_PROMPT_TEMPLATE=non-existent`
- **THEN** system logs warning and uses default template
- **AND** system continues initialization without error

### Requirement: Batch embedding requests
The system SHALL support batch embedding requests to llama.cpp for improved efficiency.

#### Scenario: Single text embedding
- **WHEN** application requests embedding for one text string
- **THEN** system sends request with single content item
- **AND** system extracts embedding from response array index 0
- **AND** system returns embedding as float64 array

#### Scenario: Batch text embeddings
- **WHEN** application requests embeddings for multiple text strings
- **THEN** system sends request with all texts in single HTTP call
- **AND** system extracts embeddings from response array sorted by index
- **AND** system returns embeddings in same order as input texts

#### Scenario: Response index validation
- **WHEN** llama.cpp returns response with non-sequential indices
- **THEN** system sorts response embeddings by index before returning
- **AND** system ensures output order matches input order

### Requirement: HTTP client configuration
The system SHALL use configurable HTTP client for llama.cpp requests.

#### Scenario: Configurable request timeout
- **WHEN** administrator sets llama.cpp configuration
- **THEN** system uses default 30-second timeout for HTTP requests
- **AND** system cancels requests exceeding timeout

#### Scenario: Connection error handling
- **WHEN** llama.cpp server is unreachable
- **THEN** system returns descriptive error including connection failure reason
- **AND** system logs error with context (endpoint, timeout)

#### Scenario: Invalid response format
- **WHEN** llama.cpp returns malformed JSON response
- **THEN** system returns error indicating invalid response format
- **AND** system includes response snippet in error message for debugging

### Requirement: Independent chat and embedding configuration
The system SHALL support independent API keys and endpoints for chat and embedding providers.

#### Scenario: Gemini chat with llama.cpp embeddings
- **WHEN** administrator configures `CHAT_PROVIDER=gemini` with Gemini credentials
- **AND** configures `EMBEDDING_PROVIDER=llamacpp` with local llama.cpp endpoint
- **THEN** system successfully initializes Gemini chat model
- **AND** system successfully initializes llama.cpp embedder
- **AND** both components operate independently with separate credentials

#### Scenario: Same provider with different API keys
- **WHEN** administrator configures `CHAT_PROVIDER=gemini` with `CHAT_API_KEY=key-1`
- **AND** configures `EMBEDDING_PROVIDER=gemini` with `EMBEDDING_API_KEY=key-2`
- **THEN** system creates separate Gemini clients for each use case
- **AND** chat operations use key-1
- **AND** embedding operations use key-2

### Requirement: Provider type validation
The system SHALL validate embedding provider configuration at startup.

#### Scenario: Invalid provider name
- **WHEN** administrator sets `EMBEDDING_PROVIDER=invalid-provider`
- **THEN** system returns validation error listing supported providers
- **AND** system fails to start with clear error message

#### Scenario: Missing required configuration
- **WHEN** administrator sets `EMBEDDING_PROVIDER=llamacpp`
- **AND** omits required `EMBEDDING_BASE_URL`
- **THEN** system returns validation error indicating missing base_url
- **AND** system fails to start with clear error message

#### Scenario: Valid configuration passes validation
- **WHEN** administrator sets all required llamacpp configuration parameters
- **THEN** system validates configuration successfully
- **AND** system completes initialization
- **AND** system logs successful embedder initialization

### Requirement: Eino framework compatibility
The system SHALL implement llama.cpp embedder compatible with Eino's embedding.Embedder interface.

#### Scenario: EmbedStrings interface
- **WHEN** application calls `embedder.EmbedStrings(ctx, []string{"text1", "text2"})`
- **THEN** system returns `[][]float64` with embeddings for each input string
- **AND** system returns error if llama.cpp request fails

#### Scenario: Error propagation
- **WHEN** llama.cpp returns HTTP error status
- **THEN** system propagates error to caller with context
- **AND** error includes HTTP status code and response body

### Requirement: Dimension validation
The system SHALL validate embedding dimension configuration matches model output.

#### Scenario: Dimension mismatch detection
- **WHEN** administrator sets `EMBEDDING_DIMENSION=768`
- **AND** llama.cpp model returns embeddings with dimension 1536
- **THEN** system logs warning about dimension mismatch
- **AND** system continues with actual returned dimension

#### Scenario: Dimension not specified
- **WHEN** administrator omits `EMBEDDING_DIMENSION` or sets to 0
- **THEN** system uses dimension from first llama.cpp response
- **AND** system logs actual dimension for reference
