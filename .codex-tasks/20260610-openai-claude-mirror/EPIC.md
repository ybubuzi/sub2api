# OpenAI Claude Mirror Epic

## Goal

Ship OpenAI <-> Claude group mirror support, preserve real upstream routing, add an admin relationship graph for users, API keys, groups, and accounts, and keep validation reproducible with the Go 1.26.3 toolchain in `D:\ENV\Go126\go\bin\go.exe`.

## Scope

- Use the Go toolchain that matches `backend/go.mod`.
- Keep existing OpenAI <-> Anthropic compatibility converters and wire them through mirror groups.
- Mirror groups can only edit model mapping; source group deletion removes mirrors.
- Admin UI shows draggable API key -> group -> account relationships.
- Admin UI supports batch API key group transfer.

## Out Of Scope

- Ent code generation for the new mirror columns.
- Mirror support for platforms other than `openai` and `anthropic`.
- New silent fallbacks, mock success paths, or hidden degradation.

## Validation Gates

- Backend compile check with 60 second timeout.
- API compatibility tests.
- Frontend typecheck and relevant tests.
- Browser verification for the admin group UI when the frontend app can run.
