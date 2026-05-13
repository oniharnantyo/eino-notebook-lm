# OpenAPI Summary — Full Field Reference

## Table of Contents

1. [POST /v1/responses — Request](#post-v1responses--request)
2. [POST /v1/responses — Response](#post-v1responses--response)
3. [POST /v1/responses/compact](#post-v1responsescompact)
4. [WebSocket Transport](#websocket-transport)
5. [Enums Reference](#enums-reference)
6. [Tool Definitions](#tool-definitions)
7. [Content Type Schemas](#content-type-schemas)
8. [Error Schemas](#error-schemas)
9. [Usage Schema](#usage-schema)

---

## POST /v1/responses — Request

### CreateResponseBody

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| `model` | string | No | — | Model identifier (e.g., `gpt-5.2`) |
| `input` | string or ItemParam[] | No | — | Conversation context. String is shorthand for a single user message. |
| `previous_response_id` | string | No | — | Chain to a prior response for multi-turn |
| `tools` | ResponsesToolParam[] | No | — | Available tool definitions |
| `tool_choice` | ToolChoiceParam | No | Default: `"auto"` | Control tool invocation mode |
| `metadata` | MetadataParam | No | Max 16 keys, key ≤64 chars, value ≤512 chars | Arbitrary key-value pairs |
| `text` | TextParam | No | — | Text output configuration |
| `temperature` | number | No | Range: 0–2 | Sampling temperature |
| `top_p` | number | No | Range: 0–1 | Nucleus sampling threshold |
| `presence_penalty` | number | No | — | Penalize repeated tokens |
| `frequency_penalty` | number | No | — | Penalize frequent tokens |
| `parallel_tool_calls` | boolean | No | Default: true | Allow parallel tool calls |
| `stream` | boolean | No | Default: false | Enable SSE streaming |
| `stream_options` | StreamOptionsParam | No | — | Stream behavior options |
| `background` | boolean | No | Default: false | Run request asynchronously |
| `max_output_tokens` | integer | No | Min: 16 | Maximum output token budget |
| `max_tool_calls` | integer | No | Min: 1 | Maximum number of tool calls |
| `reasoning` | ReasoningParam | No | — | Reasoning effort/configuration |
| `safety_identifier` | string | No | Max: 64 chars | Safety monitoring identifier |
| `prompt_cache_key` | string | No | Max: 64 chars | Cache read/write key |
| `truncation` | TruncationEnum | No | Default: `"disabled"` | Input truncation control |
| `instructions` | string | No | — | System-level model guidance |
| `store` | boolean | No | Default: true | Persist response for later retrieval |
| `service_tier` | ServiceTierEnum | No | Default: `"auto"` | Processing priority tier |
| `top_logprobs` | integer | No | Range: 0–20 | Return top N token logprobs |

### Input Types

`input` accepts either:
- **string** — shorthand for `[{type: "message", role: "user", content: [{type: "input_text", text: "..."}]}]`
- **ItemParam[]** — array of items (messages, function call outputs, etc.)

---

## POST /v1/responses — Response

### ResponseResource

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique response ID (e.g., `resp_abc123`) |
| `type` | "response" | Always `"response"` |
| `created_at` | integer | Unix timestamp (seconds) |
| `completed_at` | integer or null | Unix timestamp when completed |
| `status` | ResponseStatus | Current status |
| `incomplete_details` | IncompleteDetails or null | Reason for incompleteness |
| `model` | string | Model identifier used |
| `previous_response_id` | string or null | Previous response in chain |
| `instructions` | string or null | System instructions used |
| `output` | ItemField[] | Array of output items |
| `error` | Error or null | Error if status is `failed` |
| `tools` | Tool[] or null | Tools available during this response |
| `truncation` | TruncationEnum | Truncation strategy applied |
| `parallel_tool_calls` | boolean | Whether parallel calls were allowed |
| `text` | TextField or null | Text configuration used |
| `temperature` | number or null | Temperature used |
| `top_p` | number or null | Top-p used |
| `presence_penalty` | number or null | Presence penalty used |
| `frequency_penalty` | number or null | Frequency penalty used |
| `top_logprobs` | integer or null | Logprobs returned |
| `reasoning` | Reasoning or null | Reasoning config/output |
| `usage` | Usage | Token usage statistics |
| `max_output_tokens` | integer or null | Max tokens allowed |
| `max_tool_calls` | integer or null | Max tool calls allowed |
| `store` | boolean | Whether response was stored |
| `background` | boolean | Whether run in background |
| `service_tier` | string or null | Service tier used |
| `metadata` | object or null | Associated metadata |
| `safety_identifier` | string or null | Safety identifier |
| `prompt_cache_key` | string or null | Prompt cache key |

### ResponseStatus

| Status | Description |
|--------|-------------|
| `queued` | Waiting to be processed |
| `in_progress` | Currently being generated |
| `completed` | Successfully completed |
| `failed` | Errored out |
| `incomplete` | Stopped without completing |

### IncompleteDetails

```json
{
  "reason": "max_output_tokens_reached"
}
```

---

## POST /v1/responses/compact

Returns compacted conversation state for long-running contexts.

### Request Body

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `model` | string | Yes | Model ID for compaction |
| `input` | UserContent[] | No | Additional inputs to include |
| `previous_response_id` | string | No | Previous response to compact from |
| `system_message` | string | No | System/developer message |
| `prompt_cache_key` | string | No | Cache key |

### Response Body

```json
{
  "id": "comp_123",
  "type": "response.compaction",
  "output": [
    { "type": "message", "role": "system", "content": [...] }
  ],
  "created_at": 1234567890,
  "usage": {
    "input_tokens": 100,
    "output_tokens": 50,
    "total_tokens": 150
  }
}
```

---

## WebSocket Transport

Servers MAY expose `/v1/responses` over WebSocket as an alternative to HTTP+SSE.

### Client → Server: response.create

```json
{
  "type": "response.create",
  "model": "gpt-5.2",
  "store": false,
  "input": [
    {
      "type": "message",
      "role": "user",
      "content": "Hello"
    }
  ],
  "tools": []
}
```

### Server → Client: Streaming events

Same event objects as HTTP SSE, sent as WebSocket messages.

### Constraints

- One in-flight response per connection at a time
- Multiple requests processed sequentially
- 60-minute maximum connection duration
- On error, server sends an `error` typed message

### WebSocket-specific error codes

| Code | Description |
|------|-------------|
| `previous_response_not_found` | Referenced response ID doesn't exist |
| `websocket_connection_limit_reached` | Too many concurrent connections |

---

## Enums Reference

### MessageRole
`user` | `assistant` | `system` | `developer`

### FunctionCallStatus / MessageStatus
`in_progress` | `completed` | `incomplete`

### ToolChoiceParam
- `"auto"` — model decides
- `"none"` — no tools allowed
- `"required"` — must call at least one tool
- `{"type": "function", "name": "fn_name"}` — must call specific function
- `{"type": "allowed_tools", "tools": [...]}` — restrict to subset of tools

### TruncationEnum
- `"auto"` — server may truncate input to fit context window
- `"disabled"` — fail if input exceeds context window

### ServiceTierEnum
- `"auto"` — automatic selection
- `"default"` — standard processing
- `"flex"` — flexible/slower processing
- `"priority"` — priority processing

---

## Tool Definitions

### Function Tool

```json
{
  "type": "function",
  "name": "function_name",
  "description": "What the function does",
  "parameters": {
    "type": "object",
    "properties": {
      "param_name": {
        "type": "string",
        "description": "Parameter description"
      }
    },
    "required": ["param_name"],
    "additionalProperties": false
  },
  "strict": true
}
```

When `strict: true`, the schema is enforced exactly — no additional properties, all required fields must be present.

### Web Search Tool

```json
{
  "type": "web_search",
  "user_location": {
    "type": "approximate",
    "city": "San Francisco",
    "country": "US"
  },
  "search_context_size": "medium"
}
```

### File Search Tool

```json
{
  "type": "file_search",
  "vector_store_ids": ["vs_abc123"],
  "max_num_results": 10
}
```

### Code Interpreter Tool

```json
{
  "type": "code_interpreter",
  "container": {
    "type": "auto",
    "file_ids": ["file-abc123"]
  }
}
```

### Computer Use Tool

```json
{
  "type": "computer_preview",
  "display_width": 1024,
  "display_height": 768,
  "environment": "browser"
}
```

---

## Content Type Schemas

### Input Content Types

**InputTextContent:**
```json
{ "type": "input_text", "text": "Hello" }
```

**InputImageContent:**
```json
{
  "type": "input_image",
  "image_url": "https://example.com/image.png",
  "detail": "auto"
}
```
Alternative: `data:image/png;base64,<base64_data>` for `image_url`.

**InputFileContent:**
```json
{
  "type": "input_file",
  "file_url": "https://example.com/document.pdf",
  "filename": "document.pdf"
}
```

**InputVideoContent:**
```json
{
  "type": "input_video",
  "video_url": "https://example.com/video.mp4"
}
```

### Output Content Types

**OutputTextContent:**
```json
{
  "type": "output_text",
  "text": "Generated text here",
  "annotations": []
}
```

Annotations may include:
- `{ "type": "url_citation", "url": "...", "title": "...", "start_index": 0, "end_index": 10 }`
- `{ "type": "file_citation", "file_id": "...", "quote": "..." }`

**RefusalContent:**
```json
{ "type": "refusal", "refusal": "I cannot assist with that." }
```

**SummaryTextContent:**
```json
{ "type": "summary_text", "text": "Summary of reasoning..." }
```

---

## Error Schemas

### Error Object

```json
{
  "message": "Human-readable error description",
  "type": "error_type_string",
  "param": "field_name_or_null",
  "code": "error_code_string"
}
```

### Standard Error Types and HTTP Status Codes

| Error Type | HTTP Status | Code Examples |
|------------|-------------|---------------|
| `server_error` | 500 | `model_error`, `internal_error` |
| `invalid_request_error` | 400 | `invalid_input`, `model_not_found`, `context_length_exceeded` |
| `not_found_error` | 404 | `resource_not_found` |
| `rate_limit_error` | 429 | `rate_limit_exceeded` |
| `authentication_error` | 401 | `invalid_api_key` |

---

## Usage Schema

```json
{
  "input_tokens": 45,
  "output_tokens": 120,
  "total_tokens": 165,
  "input_tokens_details": {
    "cached_tokens": 30
  },
  "output_tokens_details": {
    "reasoning_tokens": 40
  }
}
```

All token counts are integers. `input_tokens_details` and `output_tokens_details` are optional and provide breakdowns.
