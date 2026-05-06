## MODIFIED Requirements

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
