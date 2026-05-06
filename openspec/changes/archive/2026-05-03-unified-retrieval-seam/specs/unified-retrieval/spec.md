## ADDED Requirements

### Requirement: Table Configuration
The unified retriever SHALL support multiple table types through configuration. Each table type SHALL specify its table name, BM25 index name, and optional JOIN clause for related data.

#### Scenario: Configure knowledges table
- **WHEN** retriever is configured with table type "knowledges"
- **THEN** table name is "knowledges"
- **AND** BM25 index is "knowledges_bm25_idx"
- **AND** no JOIN clause is used

#### Scenario: Configure sentences table with JOIN
- **WHEN** retriever is configured with table type "sentences"
- **THEN** table name is "sentences"
- **AND** BM25 index is "sentences_bm25_idx"
- **AND** JOIN clause includes knowledges table

#### Scenario: Configure images table
- **WHEN** retriever is configured with table type "images"
- **THEN** table name is "images"
- **AND** BM25 index is "images_bm25_idx"
- **AND** no JOIN clause is used

### Requirement: Semantic Search
The unified retriever SHALL perform vector similarity search using pgvector's cosine distance operator (`<=>`). Results SHALL be ranked by similarity score.

#### Scenario: Semantic search returns ranked documents
- **WHEN** semantic search is performed with a query vector and topK=5
- **THEN** system returns up to 5 documents
- **AND** results are ordered by cosine distance (closest first)
- **AND** each result includes document ID and rank

#### Scenario: Semantic search on unknown table type
- **WHEN** semantic search is requested for an unconfigured table type
- **THEN** system returns an error

### Requirement: Keyword Search
The unified retriever SHALL perform BM25 full-text search using PostgreSQL's pg_textsearch. Results SHALL be ranked by BM25 score in descending order.

#### Scenario: Keyword search returns BM25 ranked documents
- **WHEN** keyword search is performed with a query string and topK=5
- **THEN** system returns up to 5 documents
- **AND** results are ordered by BM25 score (highest first)
- **AND** each result includes document ID and rank

#### Scenario: Keyword search uses table-specific index
- **WHEN** keyword search is performed on sentences table
- **THEN** query uses "sentences_bm25_idx"
- **AND** search is limited to sentences table

### Requirement: Hybrid Retrieval with RRF Fusion
The unified retriever SHALL combine semantic and keyword search results using Reciprocal Rank Fusion (RRF). The top-k results from each method SHALL be fused and re-ranked.

#### Scenario: Hybrid retrieval merges results
- **WHEN** hybrid retrieval is performed with query, queryVector, topK=10, k=20
- **THEN** semantic search returns top 20 results
- **AND** keyword search returns top 20 results
- **AND** results are fused using RRF with constant k=60
- **AND** top 10 fused results are returned

#### Scenario: Hybrid retrieval fetches full documents
- **WHEN** fused results contain document IDs
- **THEN** system fetches full document content from database
- **AND** each document includes ID, content, and metadata
- **AND** metadata includes the fused RRF score

### Requirement: Backward Compatibility
The system SHALL maintain backward compatibility by providing adapter types for existing retriever interfaces. Adapters SHALL delegate to the unified retriever internally.

#### Scenario: KnowledgesRetriever adapter works
- **WHEN** code creates KnowledgesRetriever and calls Retrieve()
- **THEN** adapter delegates to unified retriever with "knowledges" table type
- **AND** results are identical to previous implementation

#### Scenario: SentencesRetriever adapter works
- **WHEN** code creates SentencesRetriever and calls Retrieve()
- **THEN** adapter delegates to unified retriever with "sentences" table type
- **AND** results include JOIN data from knowledges table

#### Scenario: ImagesRetriever adapter works
- **WHEN** code creates ImagesRetriever and calls Retrieve()
- **THEN** adapter delegates to unified retriever with "images" table type
- **AND** results are identical to previous implementation

### Requirement: Error Handling
The unified retriever SHALL return descriptive errors for invalid configurations or failed operations.

#### Scenario: Invalid dimension in config
- **WHEN** retriever is created with dimension <= 0
- **THEN** constructor returns error describing invalid dimension

#### Scenario: Nil connection pool
- **WHEN** retriever is created with nil pool
- **THEN** constructor returns error describing missing pool

#### Scenario: Database query failure
- **WHEN** database query fails during retrieval
- **THEN** error wraps the underlying database error
- **AND** error message includes context (semantic search, keyword search, etc.)