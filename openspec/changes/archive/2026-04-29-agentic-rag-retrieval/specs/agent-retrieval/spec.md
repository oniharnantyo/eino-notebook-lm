## ADDED Requirements

### Requirement: Agent-driven retrieval loop
The system SHALL use Eino's ChatModelAgent with ReAct pattern to drive retrieval iteratively. The agent SHALL reason, call retrieval tools, evaluate results, and repeat until it has sufficient information or reaches MaxIterations (30). The agent MUST NOT exceed 30 iterations per request.

#### Scenario: Agent iterates until confident
- **WHEN** a user submits a query
- **AND** agent mode is enabled
- **THEN** the system creates a ChatModelAgent with the three retrieval tools
- **AND** the agent reasons and calls tools iteratively
- **AND** the loop terminates when the LLM produces a response without tool calls or MaxIterations is reached

#### Scenario: Agent hits iteration limit
- **WHEN** the agent has performed 30 tool-calling iterations
- **THEN** the system SHALL terminate the loop
- **AND** return whatever response the agent has produced so far

### Requirement: Keyword search tool
The system SHALL provide a `keyword_search` tool that performs BM25 full-text search on the `sentences` table. The tool SHALL accept a query string and optional limit parameter (default 5). Results SHALL be scoped to the request's source IDs and source types. The tool MUST return abbreviated snippets (first 80 characters) with sentence ID, chunk ID (knowledge_id), and relevance score.

#### Scenario: Successful keyword search
- **WHEN** the agent calls `keyword_search` with query "mitochondria"
- **THEN** the system performs BM25 search on sentences matching the query
- **AND** returns up to 5 results, each containing a truncated snippet, chunk_id, and score
- **AND** results are filtered by the request's source IDs and source types

#### Scenario: Keyword search with custom limit
- **WHEN** the agent calls `keyword_search` with query "ATP" and limit 10
- **THEN** the system returns up to 10 results

#### Scenario: Keyword search finds no results
- **WHEN** the agent calls `keyword_search` with a query that matches no sentences
- **THEN** the system returns an empty result set with a message indicating no matches

### Requirement: Semantic search tool
The system SHALL provide a `semantic_search` tool that performs vector cosine similarity search on the `sentences` table embeddings. The tool SHALL accept a natural language query string and optional limit parameter (default 5). The query SHALL be embedded using the configured embedding model. Results SHALL be scoped to the request's source IDs and source types. The tool MUST return abbreviated snippets (first 80 characters) with sentence ID, chunk ID (knowledge_id), and similarity score.

#### Scenario: Successful semantic search
- **WHEN** the agent calls `semantic_search` with query "how do cells produce energy"
- **THEN** the system embeds the query and performs cosine similarity search
- **AND** returns up to 5 results, each containing a truncated snippet, chunk_id, and score
- **AND** results are filtered by the request's source IDs and source types

#### Scenario: Semantic search with custom limit
- **WHEN** the agent calls `semantic_search` with query "cell biology" and limit 10
- **THEN** the system returns up to 10 results

#### Scenario: Semantic search finds no results
- **WHEN** the agent calls `semantic_search` with a query below the score threshold
- **THEN** the system returns an empty result set with a message indicating no matches

### Requirement: Image search tool
The system SHALL provide an `image_search` tool that performs vector cosine similarity search on the `images` table vision embeddings. The tool SHALL accept a natural language query string and optional limit parameter (default 5). The query SHALL be embedded using the configured text embedding model (NOT vision embedder — text-to-image retrieval). Results SHALL be scoped to the request's source IDs and source types. The tool MUST return image metadata including s3_key, description, page_number, and similarity score.

#### Scenario: Successful image search
- **WHEN** the agent calls `image_search` with query "quarterly revenue chart"
- **THEN** the system embeds the query and performs cosine similarity search on vision embeddings
- **AND** returns up to 5 results, each containing image_id, s3_key, description, page_number, and score
- **AND** results are filtered by the request's source IDs and source types

#### Scenario: Image search with custom limit
- **WHEN** the agent calls `image_search` with query "cell structure" and limit 10
- **THEN** the system returns up to 10 results

#### Scenario: Image search finds no results
- **WHEN** the agent calls `image_search` with a query below the score threshold
- **THEN** the system returns an empty result set with a message indicating no matches

### Requirement: Chunk read tool
The system SHALL provide a `chunk_read` tool that returns the full content of a knowledge chunk given its chunk ID (knowledge_id). The tool SHALL use the KnowledgeRepository to fetch the chunk. The returned content SHALL be the full chunk text (up to ~1000 tokens as chunked during ingestion).

#### Scenario: Read an unread chunk
- **WHEN** the agent calls `chunk_read` with a chunk_id that has not been read in this request
- **THEN** the system returns the full chunk content
- **AND** marks the chunk as read in the Context Tracker

#### Scenario: Read an already-read chunk
- **WHEN** the agent calls `chunk_read` with a chunk_id already read in this request
- **THEN** the system returns a message "Chunk {id} has already been read. Explore other areas for new information."
- **AND** does NOT return the full chunk content

#### Scenario: Read a non-existent chunk
- **WHEN** the agent calls `chunk_read` with a chunk_id that does not exist
- **THEN** the system returns an error message "Chunk {id} not found"

### Requirement: Context Tracker
The system SHALL maintain a per-request Context Tracker that records every chunk read during an agent run. The tracker MUST be goroutine-safe. The tracker SHALL be created fresh for each agent request and discarded after the request completes.

#### Scenario: Tracker deduplicates reads
- **WHEN** chunk_read is called for chunk "abc-123"
- **AND** then chunk_read is called for chunk "abc-123" again in the same request
- **THEN** the first call returns full content and marks it read
- **AND** the second call returns "already read" without fetching content

#### Scenario: Tracker is request-scoped
- **WHEN** two separate agent requests are processed
- **THEN** each request has its own independent tracker
- **AND** chunks read in one request do not affect the other

### Requirement: Tool scoping via factory
The system SHALL provide a ToolFactory that creates retrieval tools scoped to a specific set of source IDs and source types. The factory SHALL inject scope parameters into the tools via closure so the agent does not need to specify them. The factory SHALL accept a ContextTracker instance per request.

#### Scenario: Server provides RAG tools automatically
- **WHEN** any request is received (with or without client-provided tools)
- **THEN** the system ALWAYS includes the four RAG retrieval tools (keyword_search, semantic_search, chunk_read, image_search)
- **AND** these tools are scoped to the request's source_ids and source_types

#### Scenario: Client provides custom tools
- **WHEN** a request includes custom tools in the tools parameter
- **THEN** the system merges client tools with server-provided RAG tools
- **AND** the agent has access to all tools (RAG + custom)

#### Scenario: Request without tools parameter
- **WHEN** a request has no tools parameter or empty tools array
- **THEN** the system still provides all four RAG retrieval tools
- **AND** the agent uses only the RAG tools

#### Scenario: Tools are scoped to request sources
- **WHEN** a request includes sourceIDs ["src-1", "src-2"]
- **THEN** all search tools created by the factory filter results to only those sources
- **AND** the agent's tool inputs do not include source filtering parameters

#### Scenario: Tools use per-request tracker
- **WHEN** the factory creates tools with a new ContextTracker
- **THEN** chunk_read uses that tracker for deduplication within the request

### Requirement: Universal agent instruction prompt
The system SHALL use a single universal instruction prompt for the agent across all notebooks. The prompt SHALL instruct the agent to use the four retrieval tools iteratively, to read chunks for full context after finding relevant snippets, to search for images when visual information may be relevant, and to provide a final answer when sufficient evidence is gathered.

#### Scenario: Agent uses the universal prompt
- **WHEN** an agent is created for any notebook
- **THEN** the same instruction prompt is used regardless of notebook identity
- **AND** the prompt describes the four available tools and the iterative retrieval strategy

### Requirement: Conversation history integration
The system SHALL inject conversation history as messages into the agent's Runner. History SHALL be loaded from the ConversationRepository, trimmed by the HistoryManager, and appended with the new user message. The agent MUST NOT have a history tool — history is ambient context.

#### Scenario: Agent sees conversation context
- **WHEN** a user sends a follow-up query with a PreviousResponseID
- **THEN** the system loads and trims the conversation history
- **AND** passes history messages + new user message to the Runner
- **AND** the agent reasons using prior conversation context

#### Scenario: First message has no history
- **WHEN** a user sends a query without a PreviousResponseID
- **THEN** the agent receives only the new user message
- **AND** operates without prior conversation context

### Requirement: Response use case always uses agent
The system SHALL use the agent for ALL response generation. The `agent_mode` request field SHALL be removed. The existing chain-based response path SHALL be deleted. All requests go through the ChatModelAgent with retrieval tools.

### Requirement: OpenResponses contract compliance
The system SHALL encode agent responses following the OpenResponses API contract (https://www.openresponses.org/reference). The ResponseResource output array SHALL contain ItemField objects representing each intermediate step.

#### Scenario: Tool call encoded as FunctionCall item
- **WHEN** the agent calls keyword_search with query "mitochondria"
- **THEN** the output array contains a FunctionCall item with type="function", name="keyword_search", and arguments JSON
- **AND** the FunctionCall includes id, call_id for tracking

#### Scenario: Tool output encoded as FunctionCallOutput item
- **WHEN** the keyword_search tool returns results
- **THEN** the output array contains a FunctionCallOutput item with type="function_output", call_id matching the FunctionCall
- **AND** the output contains the search results as JSON string

#### Scenario: Reasoning encoded as ReasoningBody item
- **WHEN** the model generates chain-of-thought reasoning (if supported by the chat model)
- **THEN** the output array contains a ReasoningBody item with type="reasoning"
- **AND** the content includes ReasoningTextContent with the reasoning text

#### Scenario: Final answer encoded as Message item
- **WHEN** the agent completes and produces a final answer
- **THEN** the output array contains a Message item with role="assistant"
- **AND** the content includes OutputTextContent with the final answer text

### Requirement: Streaming events per OpenResponses format
The system SHALL stream agent progress using OpenResponses event types via Server-Sent Events. Each event SHALL include type, sequence_number, and relevant payload.

#### Scenario: Response lifecycle events
- **WHEN** an agent request starts
- **THEN** emit `response.created` with initial ResponseResource
- **AND** emit `response.in_progress` when processing begins
- **AND** emit `response.completed` with final output when done, or `response.failed` on error

#### Scenario: Output item events
- **WHEN** the agent produces each output item (FunctionCall, FunctionCallOutput, ReasoningBody, Message)
- **THEN** emit `response.output_item.added` when the item is created
- **AND** emit `response.output_item.done` when the item is complete
- **AND** include output_index for ordering

#### Scenario: Text delta events for streaming responses
- **WHEN** the final Message content is being streamed token-by-token
- **THEN** emit `response.content_part.added` when content starts
- **AND** emit `response.output_text.delta` for each token chunk
- **AND** emit `response.output_text.done` with full text when complete

### Requirement: Usage statistics per OpenResponses format
The system SHALL populate the Usage object in ResponseResource with token counts from the agent run.

#### Scenario: Token usage includes reasoning tokens
- **WHEN** the agent completes with reasoning steps
- **THEN** Usage includes input_tokens, output_tokens, total_tokens
- **AND** output_tokens_details includes reasoning_tokens if the model exposed chain-of-thought

#### Scenario: Cached tokens tracked
- **WHEN** conversation history includes cached context
- **THEN** input_tokens_details includes cached_tokens count

#### Scenario: All requests use agent
- **WHEN** any request is received (regardless of parameters)
- **THEN** the system delegates to the ChatModelAgent
- **AND** the agent's final response is converted to ResponseResource format
- **AND** the conversation is saved for future context
- **AND** there is NO chain-based fallback path

### Requirement: Retriever method exposure
The system SHALL expose `KeywordSearch()` and `SemanticSearch()` as public methods on the existing `pgvector.Retriever`. These methods SHALL wrap the current `executeBM25SearchRanked` and `executeVectorSearch` internal methods. The original `Retrieve()` hybrid method SHALL remain unchanged.

#### Scenario: KeywordSearch returns BM25 results
- **WHEN** `KeywordSearch()` is called with a query and options
- **THEN** it performs only BM25 search without vector search or RRF merging
- **AND** returns `[]*schema.Document` with BM25 relevance scores

#### Scenario: SemanticSearch returns vector results
- **WHEN** `SemanticSearch()` is called with a query embedding and options
- **THEN** it performs only vector cosine similarity search without BM25 or RRF merging
- **AND** returns `[]*schema.Document` with distance-based scores
