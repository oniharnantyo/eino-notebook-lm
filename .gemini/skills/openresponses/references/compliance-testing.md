# Compliance Testing Guide

## Overview

The OpenResponses project includes a compliance test suite to validate that your API implementation conforms to the specification. Tests validate responses against the OpenAPI schema and semantic rules.

## Test Methods

### 1. Web UI

Interactive tester at https://www.openresponses.org/compliance

**Configuration fields:**
- **Base URL** — your API's base URL (e.g., `http://localhost:8000/v1`)
- **Model** — the model identifier your API exposes
- **API Key** — authentication key
- **Auth Header Name** — header name for the key (default: `Authorization`)
- **Use Bearer prefix** — whether to prefix the key with `Bearer ` (default: true)

### 2. CLI

```bash
bun run test:compliance --base-url http://localhost:8000/v1 --api-key $API_KEY
```

**Filter specific tests:**
```bash
bun run test:compliance \
  --base-url http://localhost:8000/v1 \
  --api-key $API_KEY \
  --filter basic-response,streaming-response
```

**All CLI flags:**
```bash
bun run test:compliance --help
```

## Test Categories

### basic-response
Validates the simplest case: a non-streaming text response.

**Checks:**
- Response has `type: "response"`
- Response has a valid `id`
- Response has `status: "completed"`
- Output contains at least one message item
- Message item has `role: "assistant"`
- Message content contains `output_text` type
- `created_at` and `completed_at` are valid Unix timestamps
- `usage` object is present with token counts

### streaming-response
Validates SSE streaming for a text response.

**Checks:**
- Response has `Content-Type: text/event-stream`
- Events follow SSE format (`data: <json>\n\n`)
- First event is `response.created`
- `response.output_item.added` event present
- `response.output_text.delta` events present
- `response.output_text.done` event present
- `response.completed` terminal event present
- Stream ends with `data: [DONE]`
- `sequence_number` values are monotonically increasing
- Event `type` matches SSE `event` field

### tool-calling
Validates function call and result flow.

**Checks:**
- Model can emit `function_call` items
- `function_call` has `call_id`, `name`, `arguments` fields
- Arguments is a valid JSON string
- `function_call_output` can be fed back via `input`
- Response after tool result contains a text message
- `call_id` in output matches the one from the function call

### multi-turn
Validates conversation chaining via `previous_response_id`.

**Checks:**
- Second request references first response's ID
- Context from first response is available in second
- Response IDs are unique
- `previous_response_id` field in response matches request

### error-handling
Validates proper error responses.

**Checks:**
- Invalid model returns `invalid_request_error` (400)
- Missing auth returns `authentication_error` (401)
- Non-existent endpoint returns `not_found_error` (404)
- Error object has `message`, `type`, and `code` fields

### item-lifecycle
Validates item status state machine.

**Checks:**
- Items start with `status: "in_progress"` in streaming
- Items transition to `status: "completed"` or `"incomplete"`
- No invalid state transitions (e.g., `completed` → `in_progress`)
- Terminal states are final (no further transitions)

## Common Failures and Fixes

### Failure: Missing `type` field in response
**Fix:** Ensure every response object includes `"type": "response"`.

### Failure: `sequence_number` not monotonically increasing
**Fix:** Use a simple counter starting at 0, incrementing by 1 for each event. Do not reset between items.

### Failure: SSE event type mismatch
**Fix:** The SSE `event:` field and the JSON `type` field must match. For example:
```
event: response.output_text.delta
data: {"type": "response.output_text.delta", ...}
```

### Failure: `function_call` missing `call_id`
**Fix:** Every `function_call` item must have a unique `call_id`. This is what links the call to its `function_call_output`.

### Failure: Item status never reaches terminal state
**Fix:** Every item created with `in_progress` must eventually emit an event transitioning it to `completed` or `incomplete`.

### Failure: Missing `[DONE]` terminal event
**Fix:** After the last `response.completed`/`response.failed`/`response.incomplete` event, send a final `data: [DONE]` line.

### Failure: `content` array empty in `output_item.done`
**Fix:** The `output_item.done` event's item must include the fully populated `content` array, not an empty one.

### Failure: Missing `usage` in response
**Fix:** Include the `usage` object with at least `input_tokens`, `output_tokens`, and `total_tokens`.

## Test Data Patterns

### Minimal valid request
```json
{
  "model": "any-model",
  "input": "Hello"
}
```

### Minimal valid response
```json
{
  "id": "resp_abc123",
  "type": "response",
  "status": "completed",
  "model": "any-model",
  "created_at": 1704067200,
  "completed_at": 1704067201,
  "output": [
    {
      "type": "message",
      "id": "msg_001",
      "role": "assistant",
      "status": "completed",
      "content": [
        { "type": "output_text", "text": "Hello!" }
      ]
    }
  ],
  "usage": {
    "input_tokens": 5,
    "output_tokens": 2,
    "total_tokens": 7
  }
}
```

### Minimal SSE stream
```
data: {"type":"response.created","sequence_number":0,"response":{"id":"resp_abc","status":"in_progress"}}

data: {"type":"response.in_progress","sequence_number":1,"response":{"id":"resp_abc","status":"in_progress"}}

data: {"type":"response.output_item.added","sequence_number":2,"output_index":0,"item":{"id":"msg_001","type":"message","status":"in_progress","role":"assistant","content":[]}}

data: {"type":"response.content_part.added","sequence_number":3,"item_id":"msg_001","output_index":0,"content_index":0,"part":{"type":"output_text","text":""}}

data: {"type":"response.output_text.delta","sequence_number":4,"item_id":"msg_001","output_index":0,"content_index":0,"delta":"Hello!"}

data: {"type":"response.output_text.done","sequence_number":5,"item_id":"msg_001","output_index":0,"content_index":0,"text":"Hello!"}

data: {"type":"response.content_part.done","sequence_number":6,"item_id":"msg_001","output_index":0,"content_index":0,"part":{"type":"output_text","text":"Hello!"}}

data: {"type":"response.output_item.done","sequence_number":7,"output_index":0,"item":{"id":"msg_001","type":"message","status":"completed","role":"assistant","content":[{"type":"output_text","text":"Hello!"}]}}

data: {"type":"response.completed","sequence_number":8,"response":{"id":"resp_abc","status":"completed","output":[{"type":"message","id":"msg_001","status":"completed","content":[{"type":"output_text","text":"Hello!"}]}]}}

data: [DONE]
```
