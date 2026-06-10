# Progress

## 2026-06-10

- Started investigation for mirror group API keys returning an unusable `/v1/models` list to Claude clients.
- Fixed mirror runtime `/v1/models` so mirror keys expose client-facing mirror mapping keys before falling back to source group models/defaults.
- Fixed admin group update handler to pass `mirror_model_mapping` into `UpdateGroupInput`; this was the root cause of existing mirror mappings not being editable after creation.
- Updated admin UI action labels so existing mirrors show "编辑映射" / "Edit Mapping" and the modal title reflects editing.
- Validation passed:
  - `D:\ENV\Go126\go\bin\go.exe test ./internal/handler/admin -run 'TestGroupHandler(UpdateMirrorModelMapping|SetMirrorCanUpdateMapping|Endpoints)' -timeout=60s`
  - `D:\ENV\Go126\go\bin\go.exe test ./internal/handler -run 'TestGatewayModels_(MirrorGroup|CustomModelsList|GeminiGroup|OpenAI)|TestMirrorRouting_AnthropicMirrorToOpenAIUsesSourceModel' -timeout=60s`
  - `pnpm typecheck`

## Recovery

任务: 修复镜像分组 Key 的运行时模型列表
形态: single-full
进度: 5/5
当前: complete
文件: `.codex-tasks/20260610-mirror-runtime-models-list/TODO.csv`
