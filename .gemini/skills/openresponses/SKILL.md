---
name: openresponses
description: Build OpenResponses-compliant LLM API servers — the open spec for multi-provider, interoperable language model interfaces based on the OpenAI Responses API. Use this skill whenever the user mentions OpenResponses, building an LLM API server, implementing the Responses API spec, multi-provider LLM interoperability, agentic tool-calling loops, or SSE streaming for LLM outputs. Also trigger when the user asks about implementing the OpenAI Responses API format, LLM API compliance testing, or building tool-calling/agent infrastructure that works across providers.
---

# OpenResponses — Implementation Skill

OpenResponses is an open specification for multi-provider, interoperable LLM interfaces based on the OpenAI Responses API. This skill guides you through building a compliant API server.

## Why This Skill Exists

LLM APIs have converged on similar building blocks (messages, tool calls, streaming, multimodal inputs) but each provider encodes them differently. OpenResponses provides a **shared, open specification** so you describe requests and outputs once, and run them across providers with minimal translation.

The spec centers on four ideas:
1. **Agentic loop** — model perceives, reasons, acts through tools, reflects on outcomes
2. **Items** — the atomic unit of context (bidirectional input/output)
3. **Semantic streaming events** — not raw text deltas, but structured lifecycle events
4. **State machines** — every object has finite states with clear transitions

## Architecture Overview

```
Client Request (POST /v1/responses)
        │
        ▼
┌─────────────────┐
│  Request Parser  │  ← Validate input, resolve previous_response_id
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Model Runner   │  ← Call LLM provider (any provider)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Item Emitter    │  ← Emit items with lifecycle states
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
 JSON      SSE Stream
 Response   Events
```

## Implementation Steps

### Step 1: Understand Items

Items are the core abstraction. Every input and output is an item.

**Required properties on every item:**
- `id` — unique opaque identifier
- `type` — discriminator for item schema
- `status` — lifecycle state

**Item status states:**

| Status | Meaning | Terminal? |
|--------|---------|-----------|
| `in_progress` | Model is currently generating this item | No |
| `completed` | Model finished successfully | Yes |
| `incomplete` | Model exhausted token budget or was interrupted | Yes |

**Standard item types:**

Message item (model output):
```json
{
  "type": "message",
  "id": "msg_01A2B3C4",
  "role": "assistant",
  "status": "completed",
  "content": [
    { "type": "output_text", "text": "Hello! How can I help?" }
  ]
}
```

Function call item (tool invocation):
```json
{
  "type": "function_call",
  "id": "fc_00123",
  "call_id": "call_987zyx",
  "name": "get_weather",
  "arguments": "{\"location\":\"Tokyo\"}",
  "status": "completed"
}
```

Function call output item (tool result fed back):
```json
{
  "type": "function_call_output",
  "call_id": "call_987zyx",
  "output": "{\"temp\":22,\"condition\":\"sunny\"}",
  "status": "completed"
}
```

Reasoning item (chain-of-thought trace):
```json
{
  "type": "reasoning",
  "status": "completed",
  "summary": [
    { "type": "summary_text", "text": "User asked about weather..." }
  ],
  "content": [
    { "type": "output_text", "text": "Need to call weather API..." }
  ],
  "encrypted_content": null
}
```

### Step 2: Implement the Request/Response Cycle

**Endpoint:** `POST /v1/responses`

**Required request headers:**
- `Authorization: Bearer <token>`
- `Content-Type: application/json`

**Key request body fields:**

| Field | Type | Required | Purpose |
|-------|------|----------|---------|
| `model` | string | Recommended | Model identifier |
| `input` | string or ItemParam[] | No | Conversation context |
| `previous_response_id` | string | No | Chain to prior response |
| `tools` | array | No | Available tool definitions |
| `tool_choice` | string/object | No | Control tool invocation |
| `stream` | boolean | No | Enable SSE streaming |
| `instructions` | string | No | System-level guidance |
| `temperature` | number (0-2) | No | Sampling randomness |
| `max_output_tokens` | integer (≥16) | No | Token budget |
| `max_tool_calls` | integer (≥1) | No | Tool call budget |
| `store` | boolean | No | Persist for retrieval |
| `metadata` | object | No | Up to 16 key-value pairs |

**Example request:**
```json
{
  "model": "gpt-5.2",
  "input": [
    { "type": "message", "role": "user", "content": "What's the weather in Tokyo?" }
  ],
  "tools": [
    {
      "type": "function",
      "name": "get_weather",
      "description": "Get current weather",
      "parameters": {
        "type": "object",
        "properties": {
          "location": { "type": "string" }
        },
        "required": ["location"]
      },
      "strict": true
    }
  ],
  "tool_choice": "auto",
  "stream": true
}
```

**Response body fields:**

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique response ID (e.g., `resp_abc123`) |
| `type` | "response" | Always `"response"` |
| `status` | string | `completed`, `failed`, `incomplete`, `in_progress`, `queued` |
| `output` | array | Array of output items |
| `model` | string | Model used |
| `created_at` | integer | Unix timestamp |
| `completed_at` | integer | Unix timestamp |
| `usage` | object | Token usage stats |
| `error` | object | Error details if failed |

**Example non-streaming response:**
```json
{
  "id": "resp_abc123",
  "type": "response",
  "status": "completed",
  "model": "gpt-5.2",
  "created_at": 1704067200,
  "completed_at": 1704067201,
  "output": [
    {
      "type": "message",
      "id": "msg_001",
      "role": "assistant",
      "status": "completed",
      "content": [
        { "type": "output_text", "text": "Tokyo is currently 22°C and sunny." }
      ]
    }
  ],
  "usage": {
    "input_tokens": 45,
    "output_tokens": 12,
    "total_tokens": 57
  }
}
```

### Step 3: Implement Streaming (SSE)

When `stream: true`, respond with `Content-Type: text/event-stream`.

**SSE format rules:**
- Each event is a JSON object prefixed with `data: `
- Events are separated by blank lines
- The terminal event is the literal string `data: [DONE]`
- The SSE `event` field MUST match the `type` field in the JSON body

**Streaming lifecycle for a single output item:**

```
response.created
response.in_progress
response.output_item.added          ← item created with status: in_progress
response.content_part.added         ← content part created
response.output_text.delta          ← (repeated) text chunks
response.output_text.done           ← final complete text
response.content_part.done          ← content part finalized
response.output_item.done           ← item finalized with status: completed
response.completed                  ← entire response done
```

**Example SSE stream:**
```
data: {"type":"response.created","response":{"id":"resp_abc","status":"in_progress"}}

data: {"type":"response.output_item.added","output_index":0,"item":{"id":"msg_001","type":"message","status":"in_progress","content":[]}}

data: {"type":"response.content_part.added","item_id":"msg_001","output_index":0,"content_index":0,"part":{"type":"output_text","text":""}}

data: {"type":"response.output_text.delta","item_id":"msg_001","output_index":0,"content_index":0,"delta":"Tokyo"}

data: {"type":"response.output_text.delta","item_id":"msg_001","output_index":0,"content_index":0,"delta":" is 22°C"}

data: {"type":"response.output_text.done","item_id":"msg_001","output_index":0,"content_index":0,"text":"Tokyo is 22°C and sunny."}

data: {"type":"response.content_part.done","item_id":"msg_001","output_index":0,"content_index":0,"part":{"type":"output_text","text":"Tokyo is 22°C and sunny."}}

data: {"type":"response.output_item.done","output_index":0,"item":{"id":"msg_001","type":"message","status":"completed","content":[{"type":"output_text","text":"Tokyo is 22°C and sunny."}]}}

data: {"type":"response.completed","response":{"id":"resp_abc","status":"completed","output":[...]}}

data: [DONE]
```

Every streaming event includes a `sequence_number` (monotonically increasing integer) for ordering.

For the complete streaming event catalog, read `references/streaming-events.md`.

### Step 4: Implement Tool Calling (Agentic Loop)

The agentic loop is the core pattern: the model reasons, decides to call tools, receives results, and continues.

**Tool definition format:**
```json
{
  "type": "function",
  "name": "send_email",
  "description": "Send an email to a recipient",
  "parameters": {
    "type": "object",
    "properties": {
      "recipient": { "type": "string", "description": "Email address" },
      "subject": { "type": "string" },
      "body": { "type": "string" }
    },
    "required": ["recipient", "subject", "body"]
  },
  "strict": true
}
```

**Tool choice modes:**

| Value | Behavior |
|-------|----------|
| `"auto"` | Model decides (default) |
| `"required"` | Model MUST call at least one tool |
| `"none"` | Model MUST NOT call tools |
| `{"type":"function","name":"fn"}` | Must call specific function |

**Agentic loop flow:**

```
1. Client sends request with tools + input
2. Server forwards to model
3. Model returns function_call item(s)
4. Server emits function_call items to client
5. Client executes the tool locally
6. Client sends new request with:
   - previous_response_id set to the last response ID
   - input containing function_call_output item(s)
7. Repeat until model returns a message (no more tool calls)
```

**Loop example — turn 1 (model calls tool):**
```json
{
  "model": "gpt-5.2",
  "input": "What's the weather in Tokyo?",
  "tools": [{"type":"function","name":"get_weather","parameters":{...}}],
  "stream": false
}
```

Response:
```json
{
  "id": "resp_001",
  "status": "completed",
  "output": [
    {
      "type": "function_call",
      "id": "fc_001",
      "call_id": "call_abc",
      "name": "get_weather",
      "arguments": "{\"location\":\"Tokyo\"}",
      "status": "completed"
    }
  ]
}
```

**Loop example — turn 2 (client feeds tool result back):**
```json
{
  "model": "gpt-5.2",
  "previous_response_id": "resp_001",
  "input": [
    {
      "type": "function_call_output",
      "call_id": "call_abc",
      "output": "{\"temp\":22,\"condition\":\"sunny\"}"
    }
  ]
}
```

Response:
```json
{
  "id": "resp_002",
  "status": "completed",
  "output": [
    {
      "type": "message",
      "id": "msg_002",
      "role": "assistant",
      "status": "completed",
      "content": [
        { "type": "output_text", "text": "Tokyo is currently 22°C and sunny." }
      ]
    }
  ]
}
```

### Step 5: Implement Error Handling

**Error response format:**
```json
{
  "error": {
    "message": "The requested model 'fake-model' does not exist.",
    "type": "invalid_request_error",
    "param": "model",
    "code": "model_not_found"
  }
}
```

**Standard error types:**

| Type | HTTP Status | When |
|------|-------------|------|
| `server_error` | 500 | Internal failure |
| `invalid_request_error` | 400 | Malformed request |
| `not_found_error` | 404 | Resource not found |
| `model_error` | 500 | Model processing failure |
| `rate_limit_error` | 429 | Too many requests |

### Step 6: Extensibility (Custom Types)

Custom items, events, and tools use a prefix namespace to avoid collisions:

**Custom item type:**
```json
{
  "type": "acme:search_result",
  "id": "sr_123",
  "status": "completed",
  "request_id": "req_123"
}
```

**Custom streaming event:**
```json
{
  "type": "acme:trace_event",
  "sequence_number": 1,
  "phase": "tool_resolution",
  "latency_ms": 34
}
```

**Custom tool:**
```json
{
  "type": "acme:document_search",
  "documents": [
    {"type": "external_file", "url": "https://example.com/doc.pdf"}
  ]
}
```

### Step 7: Validate with Compliance Tests

After implementation, validate your API against the spec.

**Web UI:** https://www.openresponses.org/compliance

**CLI:**
```bash
bun run test:compliance --base-url http://localhost:8000/v1 --api-key $API_KEY
```

Filter specific tests:
```bash
bun run test:compliance --base-url http://localhost:8000/v1 --api-key $API_KEY --filter basic-response,streaming-response
```

For the full compliance testing guide, read `references/compliance-testing.md`.

## Content Types Reference

### Input Content Types (User → Model)
| Type | Description |
|------|-------------|
| `input_text` | Plain text input |
| `input_image` | Image (base64 or URL) |
| `input_file` | File attachment |
| `input_video` | Video input |

### Output Content Types (Model → User)
| Type | Description |
|------|-------------|
| `output_text` | Generated text |
| `refusal` | Model declined to answer |
| `summary_text` | Reasoning summary |

### Message Roles
| Role | Purpose |
|------|---------|
| `user` | End-user input |
| `assistant` | Model output |
| `system` | System instructions |
| `developer` | Developer guidance |

## Additional Endpoints

### POST /v1/responses/compact

Compacts conversation state for long-running contexts. Returns a summarized representation of the conversation that can be used as input for continuation.

```json
// Request
{
  "model": "gpt-5",
  "input": [...],
  "previous_response_id": "resp_prev",
  "system_message": "You are a helpful assistant"
}

// Response
{
  "id": "comp_123",
  "type": "response.compaction",
  "output": [...],
  "created_at": 1234567890,
  "usage": { "input_tokens": 100, "output_tokens": 50 }
}
```

### WebSocket Transport

Servers MAY expose `/v1/responses` over WebSocket:

- Client sends `response.create` messages
- One in-flight response per connection
- Same streaming event objects as HTTP
- 60-minute connection limit

## Reference Files

- **`references/openapi-summary.md`** — Full API field reference with all request/response parameters, enums, and constraints
- **`references/streaming-events.md`** — Complete streaming event catalog with all event types, their fields, and ordering
- **`references/compliance-testing.md`** — How to run compliance tests, interpret results, and fix common failures

Read these when you need the complete surface area of the spec beyond the implementation guidance above.

## Key Design Decisions to Keep in Mind

1. **Items are bidirectional** — the same item types appear in both input and output. A `function_call_output` from the client becomes part of the next request's input.

2. **`previous_response_id` chains context** — this is how you build multi-turn conversations without resending the full history. The server resolves the prior context.

3. **Streaming events are semantic** — clients don't parse raw text; they react to structured lifecycle events (`response.output_item.added`, `response.output_text.delta`, etc.).

4. **`sequence_number` guarantees ordering** — every streaming event carries a monotonically increasing sequence number. Use it for ordering, not gaps.

5. **`call_id` links tool calls to results** — when the model emits a `function_call` with `call_id: "call_abc"`, the client must return the result with a `function_call_output` referencing the same `call_id`.

6. **Status is a state machine** — items go from `in_progress` → `completed` or `incomplete`. Never jump states or go backwards.
