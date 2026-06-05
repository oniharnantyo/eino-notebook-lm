## MODIFIED Requirements

### Requirement: Semantic Search
The unified retriever SHALL perform vector similarity search using pgvector's cosine distance operator (`<=>`). Results SHALL be ranked by similarity score. When source IDs are provided, results SHALL be filtered to only include rows matching those source IDs.

#### Scenario: Semantic search returns ranked documents
- **WHEN** semantic search is performed with a query vector and topK=5
- **THEN** system returns up to 5 documents
- **AND** results are ordered by cosine distance (closest first)
- **AND** each result includes document ID and rank

#### Scenario: Semantic search on unknown table type
- **WHEN** semantic search is requested for an unconfigured table type
- **THEN** system returns an error

#### Scenario: Semantic search with source scoping on knowledges table
- **WHEN** semantic search is performed on "knowledges" table with sourceIDs [src-1, src-2]
- **THEN** SQL query includes `WHERE source_id IN ('src-1', 'src-2')`
- **AND** results only contain documents belonging to those sources

#### Scenario: Semantic search with source scoping on sentences table
- **WHEN** semantic search is performed on "sentences" table with sourceIDs [src-1]
- **THEN** SQL query includes `WHERE metadata->>'source_id' IN ('src-1')`
- **AND** results only contain sentences belonging to that source

#### Scenario: Semantic search with source scoping on images table
- **WHEN** semantic search is performed on "images" table with sourceIDs [src-1, src-2]
- **THEN** SQL query includes `WHERE source_id IN ('src-1', 'src-2')`
- **AND** results only contain images belonging to those sources

#### Scenario: Semantic search without source scoping
- **WHEN** semantic search is performed with nil or empty sourceIDs
- **THEN** no source filter is applied to the SQL query
- **AND** results span all sources in the database

### Requirement: Semantic Search Aggregated
The unified retriever SHALL perform aggregated vector similarity search on the sentences table, grouping results by `knowledge_id`. Each group SHALL use the MAX similarity score from its best-matching sentence. The best-matching sentence content SHALL be included as a snippet.

#### Scenario: Aggregated search returns chunk-level results
- **WHEN** aggregated semantic search is performed with a query vector and topK=5
- **THEN** system returns up to 5 unique knowledge_id results
- **AND** each result includes knowledge_id as the document ID
- **AND** each result includes the best-matching sentence content as snippet
- **AND** each result's score is the MAX similarity across all sentences in that group

#### Scenario: Aggregated search with source scoping
- **WHEN** aggregated search is performed with sourceIDs [src-1, src-2]
- **THEN** SQL query includes `WHERE metadata->>'source_id' IN ('src-1', 'src-2')`
- **AND** results only contain chunks from those sources

### Requirement: Keyword Search
The unified retriever SHALL perform BM25 full-text search using PostgreSQL's pg_textsearch. Results SHALL be ranked by BM25 score in descending order. When source IDs are provided, results SHALL be filtered to only include rows matching those source IDs.

#### Scenario: Keyword search returns BM25 ranked documents
- **WHEN** keyword search is performed with a query string and topK=5
- **THEN** system returns up to 5 documents
- **AND** results are ordered by BM25 score (highest first)
- **AND** each result includes document ID and rank

#### Scenario: Keyword search uses table-specific index
- **WHEN** keyword search is performed on sentences table
- **THEN** query uses "sentences_bm25_idx"
- **AND** search is limited to sentences table

#### Scenario: Keyword search with source scoping on knowledges table
- **WHEN** keyword search is performed on "knowledges" table with sourceIDs [src-1, src-2]
- **THEN** SQL query includes `WHERE source_id IN ('src-1', 'src-2')`
- **AND** results only contain documents belonging to those sources

#### Scenario: Keyword search with source scoping on sentences table
- **WHEN** keyword search is performed on "sentences" table with sourceIDs [src-1]
- **THEN** SQL query includes `WHERE metadata->>'source_id' IN ('src-1')`
- **AND** results only contain sentences belonging to that source

#### Scenario: Keyword search without source scoping
- **WHEN** keyword search is performed with nil or empty sourceIDs
- **THEN** no source filter is applied to the SQL query
- **AND** results span all sources in the database
