# Mirror Runtime Models List

## Goal

Fix runtime `/v1/models` responses for API keys bound to OpenAI/Anthropic mirror groups so Claude/OpenAI clients can discover the client-facing mirror models instead of receiving an unavailable or empty model list.

## Scope

- Inspect gateway `/v1/models` handling and mirror group routing behavior.
- Make mirror groups expose the correct target-platform model IDs, using mirror model mapping and source group availability where appropriate.
- Add focused backend tests for mirror group model list behavior.
- Run targeted validation with the configured Go toolchain.

## Out of Scope

- Frontend mirror modal candidate loading.
- Deployment changes unless explicitly requested.
