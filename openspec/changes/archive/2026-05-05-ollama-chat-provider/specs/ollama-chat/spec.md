## ADDED Requirements

### Requirement: Ollama chat model creation
The system SHALL create an Ollama chat model when `CHAT_PROVIDER` is set to `ollama`. The model SHALL implement the `ToolCallingChatModel` interface, supporting Generate, Stream, and tool calling.

#### Scenario: Create Ollama chat model with default URL
- **WHEN** `CHAT_PROVIDER=ollama` and `CHAT_MODEL=qwen3:8b` with no `CHAT_BASE_URL`
- **THEN** the system creates a chat model connected to `http://localhost:11434`
- **AND** the model is ready for Generate, Stream, and tool calling

#### Scenario: Create Ollama chat model with custom URL
- **WHEN** `CHAT_PROVIDER=ollama` and `CHAT_BASE_URL=http://192.168.1.100:11434`
- **THEN** the system creates a chat model connected to the specified URL

#### Scenario: Ollama chat model supports tool calling
- **WHEN** the retrieval agent binds tools via `WithTools()`
- **THEN** the Ollama model returns tool calls in its responses
- **AND** the agent can execute retrieval tools iteratively

### Requirement: Chat request timeout
The system SHALL support a configurable `CHAT_TIMEOUT` environment variable that sets the HTTP request timeout for chat model calls. When not set, the system SHALL use a sensible default.

#### Scenario: Custom timeout is applied
- **WHEN** `CHAT_TIMEOUT=10m` is configured
- **THEN** the chat model HTTP client uses a 10-minute timeout

#### Scenario: Default timeout when not configured
- **WHEN** `CHAT_TIMEOUT` is not set
- **THEN** the chat model uses its default timeout behavior

### Requirement: Ollama model keep-alive
The system SHALL support a configurable `CHAT_KEEP_ALIVE` environment variable that controls how long Ollama keeps the model loaded in memory between requests. This setting SHALL only apply when using the Ollama provider.

#### Scenario: Keep-alive keeps model warm
- **WHEN** `CHAT_KEEP_ALIVE=30m` is configured with Ollama provider
- **THEN** the Ollama model stays loaded for 30 minutes after the last request

#### Scenario: Keep-alive not configured
- **WHEN** `CHAT_KEEP_ALIVE` is not set with Ollama provider
- **THEN** the Ollama default keep-alive behavior applies (5 minutes)

#### Scenario: Keep-alive ignored for other providers
- **WHEN** `CHAT_KEEP_ALIVE=30m` is configured with Gemini provider
- **THEN** the keep-alive setting is ignored
