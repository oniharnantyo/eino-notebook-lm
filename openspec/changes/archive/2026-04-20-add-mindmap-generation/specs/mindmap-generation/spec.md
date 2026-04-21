## ADDED Requirements

### Requirement: Trigger mindmap generation
The system SHALL expose `POST /api/v1/notebooks/{notebookId}/sources/{sourceId}/mindmap` that triggers asynchronous mindmap generation. The endpoint SHALL validate that the source exists and has content (status = `completed`). On success, it SHALL create an artifact record with type `mindmap` and status `pending`, then return the artifact immediately while processing continues in the background.

#### Scenario: Successful trigger
- **WHEN** `POST /api/v1/notebooks/{nbId}/sources/{srcId}/mindmap` is called with a valid, completed source
- **THEN** the system creates an artifact with type `mindmap` and status `pending`, returns 202 Accepted with the artifact ID and status URL, and begins async processing

#### Scenario: Source not found
- **WHEN** the source ID does not exist
- **THEN** the system returns 404 Not Found

#### Scenario: Source has no content
- **WHEN** the source exists but has empty content
- **THEN** the system returns 422 Unprocessable Entity with message "source has no content"

#### Scenario: Source not completed
- **WHEN** the source exists but status is not `completed`
- **THEN** the system returns 409 Conflict with message "source is not ready for mindmap generation"

### Requirement: Mindmap generation process
The system SHALL generate a mindmap by sending `Source.Content` to the configured LLM (Gemini) with a structured prompt requesting a hierarchical tree JSON. On success, the artifact result SHALL be updated with the tree JSON and status set to `completed`. On failure, the artifact error SHALL be set and status set to `failed`.

#### Scenario: Successful generation
- **WHEN** the LLM returns valid JSON matching the expected tree structure
- **THEN** the artifact result is set to the parsed JSON, status is `completed`, and UpdatedAt is refreshed

#### Scenario: LLM returns malformed JSON
- **WHEN** the LLM response cannot be parsed as valid JSON or does not match the expected structure
- **THEN** the artifact status is set to `failed` with error "failed to parse LLM response as mindmap"

#### Scenario: LLM call fails
- **WHEN** the LLM API returns an error or times out
- **THEN** the artifact status is set to `failed` with the error message

#### Scenario: Panic during processing
- **WHEN** the async goroutine panics
- **THEN** the panic is recovered, artifact status is set to `failed` with panic message, and the error is logged

### Requirement: Mindmap output structure
The mindmap result JSON SHALL be a hierarchical tree where each node has: `id` (string), `label` (string), `summary` (string, 1-2 lines), and `children` (array of child nodes). The root node SHALL have a `title` field instead of `label`. All `id` values SHALL be unique within the tree.

#### Scenario: Valid tree structure
- **WHEN** a mindmap is successfully generated
- **THEN** the result JSON conforms to:
  ```json
  {
    "title": "Document Title",
    "children": [
      {
        "id": "n1",
        "label": "1. Introduction",
        "summary": "Brief summary of this section",
        "children": [...]
      }
    ]
  }
  ```

#### Scenario: Unique node IDs
- **WHEN** a mindmap result is validated
- **THEN** all `id` values across the tree are unique

### Requirement: Mindmap generation prompt
The system SHALL use a system prompt that instructs the LLM to: analyze the document content, identify main sections and subsections, generate a hierarchical tree with concise labels and 1-2 line summaries, and return ONLY valid JSON matching the required structure.

#### Scenario: Prompt produces structured output
- **WHEN** the prompt is sent with document content
- **THEN** the LLM returns a valid JSON tree representing the document's structure and key concepts

### Requirement: Mindmap uses configured chat model
The mindmap generation use case SHALL receive a `model.BaseChatModel` via constructor injection (same pattern as ResponseUseCase). It SHALL use the configured model (e.g., `gemini/gemini-2.0-flash-exp`) for generation.

#### Scenario: Uses injected chat model
- **WHEN** mindmap generation is triggered
- **THEN** the use case calls `chatModel.Generate()` with the constructed prompt and document content
