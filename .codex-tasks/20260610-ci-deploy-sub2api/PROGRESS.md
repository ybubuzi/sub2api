# Progress

## 2026-06-10

- Pushed fd6caf04 to origin/main.
- GitHub Docker Build & Push succeeded for fd6caf04.
- GitHub CI failed in backend lint/unit gates; server mutation is paused until CI is fixed.
- Fixed CI failures locally:
  - `go test -tags=unit ./internal/repository ./internal/service ./internal/server -run 'TestAPIKeyRepository_GetByKeyForAuth_PreservesMessagesDispatchModelConfig_SQLite|TestAPIKeyRepository_CreateWithLastUsedAt|TestAPIKeyRepository_UpdateLastUsed|TestAdminService_DeleteGroup|TestAPIContracts' -count=1 -timeout=60s`
  - `golangci-lint run --timeout=30m`
  - `git diff --check`
- Next step is pushing the CI fix commit and waiting for GitHub workflows.
