## MODIFIED Requirements

### Requirement: Keyword search tool
The system SHALL provide a `keyword_search` tool that performs BM25 full-text search on the `knowledges` table. The tool SHALL accept a `keywords` array of search terms and an optional `top_k` parameter (default 5). Keywords SHALL be joined with spaces into a single BM25 query string. Results SHALL be scoped to the request's source IDs and source types. The tool MUST return keyword-in-context (KWIC) snippets with ±80 characters around each keyword match, grouped by chunk ID, instead of first-80-chars truncation. Each result SHALL contain the chunk_id and an array of KWIC snippet strings.

#### Scenario: Successful keyword search with multiple keywords
- **WHEN** the agent calls `keyword_search` with keywords ["telephone", "invented"] and top_k 5
- **THEN** the system joins keywords as "telephone invented" and performs BM25 search on the knowledges table
- **AND** returns up to 5 chunk results, each containing a chunk_id and KWIC snippets showing where keywords matched
- **AND** results are filtered by the request's source IDs and source types

#### Scenario: Keyword search with multi-word phrase
- **WHEN** the agent calls `keyword_search` with keywords ["Alexander Graham Bell", "born"]
- **THEN** the system joins as "Alexander Graham Bell born" for BM25 search
- **AND** KWIC snippets show context around both the full phrase "Alexander Graham Bell" and the term "born"

#### Scenario: Keyword search with custom top_k
- **WHEN** the agent calls `keyword_search` with keywords ["energy"] and top_k 10
- **THEN** the system returns up to 10 chunk results

#### Scenario: Keyword search finds no results
- **WHEN** the agent calls `keyword_search` with keywords that match no chunks
- **THEN** the system returns an empty result set

#### Scenario: Keyword search defaults top_k to 5
- **WHEN** the agent calls `keyword_search` with keywords ["photosynthesis"] and no top_k
- **THEN** the system returns up to 5 results

### Requirement: Semantic search tool
The system SHALL provide a `semantic_search` tool that performs vector cosine similarity search on the `sentences` table embeddings. The tool SHALL accept a natural language query string and an optional `top_k` parameter (default 5). The query SHALL be embedded using the configured embedding model. Results SHALL be scoped to the request's source IDs and source types. The tool MUST return abbreviated snippets with chunk ID (knowledge_id) and similarity score.

#### Scenario: Successful semantic search
- **WHEN** the agent calls `semantic_search` with query "how do cells produce energy"
- **THEN** the system embeds the query and performs cosine similarity search
- **AND** returns up to 5 results, each containing a snippet, chunk_id, and score
- **AND** results are filtered by the request's source IDs and source types

#### Scenario: Semantic search with custom top_k
- **WHEN** the agent calls `semantic_search` with query "cell biology" and top_k 10
- **THEN** the system returns up to 10 results

#### Scenario: Semantic search finds no results
- **WHEN** the agent calls `semantic_search` with a query below the score threshold
- **THEN** the system returns an empty result set

#### Scenario: Semantic search defaults top_k to 5
- **WHEN** the agent calls `semantic_search` with query "mitochondria" and no top_k
- **THEN** the system returns up to 5 results

### Requirement: Chunk read tool
The system SHALL provide a `chunk_read` tool that returns the full content of knowledge chunks given an array of chunk IDs. The tool SHALL accept `chunk_ids` as an array of strings. The tool SHALL use the KnowledgeRepository to fetch chunks. For each requested ID, if the chunk has already been read in this request (per ContextTracker), the tool SHALL return a status message instead of content. For unread IDs, the tool SHALL fetch and return the full chunk content and mark each as read.

#### Scenario: Read multiple unread chunks
- **WHEN** the agent calls `chunk_read` with chunk_ids ["id-1", "id-2", "id-3"]
- **AND** none have been read in this request
- **THEN** the system returns an array of 3 chunks, each with full content
- **AND** marks all 3 as read in the ContextTracker

#### Scenario: Read with some already-read chunks
- **WHEN** the agent calls `chunk_read` with chunk_ids ["id-1", "id-2", "id-3"]
- **AND** "id-2" was already read in this request
- **THEN** the system returns full content for "id-1" and "id-3"
- **AND** returns a status "Chunk id-2 has already been read" for "id-2"
- **AND** marks "id-1" and "id-3" as read

#### Scenario: Read a non-existent chunk
- **WHEN** the agent calls `chunk_read` with chunk_ids ["id-1", "nonexistent"]
- **THEN** the system returns full content for "id-1"
- **AND** returns a status "Chunk nonexistent not found" for the non-existent ID

#### Scenario: Read empty array
- **WHEN** the agent calls `chunk_read` with an empty chunk_ids array
- **THEN** the system returns an empty result set

### Requirement: Tool scoping via factory
The system SHALL provide a ToolFactory that creates retrieval tools scoped to a specific set of source IDs and source types. The factory SHALL accept a knowledge retriever (for keyword_search on knowledges table), a sentence retriever (for semantic_search on sentences table), an image retriever, a knowledge repository (for chunk_read), an embedder, and a ContextTracker per request. The factory SHALL inject scope parameters into the tools via closure so the agent does not need to specify them.

#### Scenario: Server provides RAG tools automatically
- **WHEN** any request is received (with or without client-provided tools)
- **THEN** the system ALWAYS includes the four RAG retrieval tools (keyword_search, semantic_search, chunk_read, image_search)
- **AND** keyword_search uses the knowledge retriever (knowledges table)
- **AND** semantic_search uses the sentence retriever (sentences table)

#### Scenario: Tools are scoped to request sources
- **WHEN** a request includes sourceIDs ["src-1", "src-2"]
- **THEN** all search tools created by the factory filter results to only those sources
- **AND** the agent's tool inputs do not include source filtering parameters

#### Scenario: Tools use per-request tracker
- **WHEN** the factory creates tools with a new ContextTracker
- **THEN** chunk_read uses that tracker for deduplication within the request
