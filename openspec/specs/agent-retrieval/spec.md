# Capability: Agent Retrieval

## Purpose
The Agent Retrieval capability provides an iterative, agent-driven approach to information retrieval and response generation. It leverages a ChatModelAgent with a ReAct pattern to reason about user queries and call specialized retrieval tools (keyword search, semantic search, image search, and chunk reading) to gather necessary context before producing a final answer. This ensures high-quality, evidence-based responses compliant with the OpenResponses API contract.

## Requirements

### Requirement: Agent-driven retrieval loop
The system SHALL use Eino's ChatModelAgent with ReAct pattern to drive retrieval iteratively. The agent SHALL reason, call retrieval tools, evaluate results, and repeat until it has sufficient information or reaches MaxIterations (30). The agent MUST NOT exceed 30 iterations per request. The chat model SHALL be selected from the configured provider (gemini, openai, or ollama) via the `CreateToolCallingChatModel` factory.

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

#### Scenario: Agent uses Ollama chat model
- **WHEN** `CHAT_PROVIDER=ollama` is configured
- **THEN** the agent uses an Ollama-backed chat model
- **AND** tool calling works through Ollama's native tool API

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
The system SHALL provide a ToolFactory that creates retrieval tools scoped to a specific set of source IDs and source types. The factory SHALL accept a knowledge retriever (for keyword_search on knowledges table), a sentence retriever (for semantic_search on sentences table), an image retriever, a knowledge repository (for chunk_read), an embedder, and a ContextTracker per request. The factory SHALL inject scope parameters into the tools via closure so the agent does not need to specify them.

#### Scenario: Server provides RAG tools automatically
- **WHEN** any request is received (with or without client-provided tools)
- **THEN** the system ALWAYS includes the four RAG retrieval tools (keyword_search, semantic_search, chunk_read, image_search)
- **AND** keyword_search uses the knowledge retriever (knowledges table)
- **AND** semantic_search uses the sentence retriever (sentences table)

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
The system SHALL use a single universal instruction prompt template for the agent across all notebooks. The prompt SHALL instruct the agent to use the four retrieval tools iteratively, to read chunks for full context after finding relevant snippets, to search for images when visual information may be relevant, and to provide a final answer when sufficient evidence is gathered. The prompt template SHALL contain a `{catalog}` placeholder that is replaced per-request with the available sources catalog.

#### Scenario: Agent uses the universal prompt template
- **WHEN** an agent is created for any notebook
- **THEN** the same instruction prompt template is used regardless of notebook identity
- **AND** the prompt template contains a `{catalog}` placeholder for dynamic source catalog injection
- **AND** the prompt describes the four available tools and the iterative retrieval strategy

#### Scenario: Catalog placeholder is replaced per-request
- **WHEN** the agent executes a query with specific source IDs
- **THEN** the `{catalog}` placeholder is replaced with the formatted catalog string
- **AND** the catalog is passed via ADK session values as `map[string]any{"catalog": catalog}`
- **AND** the agent receives the complete system prompt with catalog injected

#### Scenario: Agent receives only selected sources in catalog
- **WHEN** a user selects specific sources for a query
- **THEN** the catalog includes only those selected sources with their IDs, titles, and status
- **AND** the agent is aware of only those sources for tool execution

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
- **WHEN** the agent calls keyword_search with keywords ["mitochondria"]
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
