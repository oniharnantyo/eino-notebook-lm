## Context

The project uses a factory pattern for chat model creation (`CreateToolCallingChatModel`) with two providers: Gemini and OpenAI. Ollama is already integrated as an embedding provider using the same pattern. The Eino framework provides a dedicated `eino-ext/components/model/ollama` component (v0.1.9) that implements `ToolCallingChatModel` — the same interface the factory returns.

Current `ChatConfig` has no timeout or keep-alive fields. The embedding side already has `Timeout` in its config.

## Goals / Non-Goals

**Goals:**
- Add Ollama as a chat provider with minimal code changes
- Make chat request timeout configurable for all providers
- Expose Ollama's KeepAlive setting to control model memory retention
- Stream responses through to the user (already supported by the Ollama component)

**Non-Goals:**
- Adding Ollama-specific tuning options (temperature, top_p, seed) to config — can be added later
- Changing the retrieval agent architecture or tool definitions
- Supporting Ollama vision/multimodal chat (out of scope for now)
- Adding health checks or model availability detection

## Decisions

**1. Use the Eino Ollama component directly (not OpenAI-compatible endpoint)**

Ollama exposes an OpenAI-compatible API, so we could reuse the OpenAI provider with `CHAT_BASE_URL=http://localhost:11434/v1`. However, the dedicated Eino Ollama component provides:
- Native tool calling via Ollama's own API (not the OpenAI compatibility layer)
- Thinking/reasoning content support for models like QwQ/DeepSeek-R1
- `KeepAlive` control for model memory management
- Callback integration with Langfuse observability

**2. Add Timeout and KeepAlive to ChatConfig (not Ollama-specific config)**

`Timeout` is broadly useful (Gemini/OpenAI could also use it later). `KeepAlive` is Ollama-specific but placing it in `ChatConfig` keeps the config path simple — other providers just ignore it. This avoids adding a second config struct or provider-specific config map.

**3. Default BaseURL to `http://localhost:11434`**

Matches the Ollama default and the existing embedding behavior. Users only need `CHAT_PROVIDER=ollama` and `CHAT_MODEL=<model>` to get started.

## Risks / Trade-offs

- **[Tool calling quality varies by model]** → Mitigation: Document that users should pick tool-calling capable models (Qwen2.5/3, Llama3.1+). The factory doesn't validate model capabilities — errors surface at runtime if the model can't handle tools.
- **[CPU inference is slow]** → Mitigation: Configurable timeout defaults to a generous value. Users can tune `CHAT_TIMEOUT` per their hardware.
- **[KeepAlive field on ChatConfig is Ollama-specific]** → Acceptable tradeoff. Other providers ignore it. Keeps config simple.
