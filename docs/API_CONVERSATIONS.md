# Conversation API

## Get Messages

Retrieves a paginated list of messages for a conversation within a notebook.

**Endpoint:** `GET /api/v1/notebooks/{notebookId}/conversations/messages`

**Query Parameters:**
- `conversation_id` (optional): The ID of the conversation. If omitted, the latest conversation for the notebook is used.
- `limit` (optional): The maximum number of messages to return. Defaults to 50. Maximum is 100.
- `before_sequence` (optional): The sequence number used as a cursor. Returns messages older than this sequence.

**Response:**

```json
{
  "messages": [
    {
      "id": "msg-uuid",
      "conversation_id": "conv-uuid",
      "sequence_num": 2,
      "response_id": "resp-uuid",
      "message": {
        "role": "assistant",
        "content": "Hello!"
      },
      "model": "gemini-pro",
      "finish_reason": "stop",
      "prompt_tokens": 10,
      "completion_tokens": 5,
      "total_tokens": 15,
      "created_at": "2023-10-01T12:00:00Z"
    }
  ],
  "conversation_id": "conv-uuid",
  "has_more": false,
  "oldest_sequence": 2
}
```

## List Conversations

Retrieves a paginated list of conversations for a notebook.

**Endpoint:** `GET /api/v1/notebooks/{notebookId}/conversations`

**Query Parameters:**
- `page` (optional): Page number, defaults to 1.
- `limit` (optional): Items per page, defaults to 10.

**Response:**

```json
{
  "conversations": [
    {
      "id": "conv-uuid",
      "notebook_id": "nb-uuid",
      "created_at": "2023-10-01T12:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "limit": 10,
  "total_pages": 1
}
```