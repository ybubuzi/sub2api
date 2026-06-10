# Progress

## 2026-06-10

- Started task record for mirror model candidate selection.
- Confirmed existing backend candidate source: `GetGroupModelsListCandidates` returns default platform models plus schedulable account `model_mapping` keys for the requested group/platform. Those keys are the public/external model names.
- Added `groupMirrorModels.ts` for mirror candidate context and candidate normalization/merge logic.
- Added `useGroupMirrorModelCandidates.ts` to load target-platform and source-platform candidate lists from the source group.
- Updated `GroupMirrorModal.vue` to use `datalist` inputs for both sides of mirror mapping while preserving manual typing.
- Updated mirror mapping copy so the right side is described as the source group's public model, not an internal upstream model.
- Validation passed:
  - `pnpm test:run src/components/admin/group/__tests__/groupMirrorModels.spec.ts src/components/admin/group/__tests__/useGroupMirrorModelCandidates.spec.ts`
  - `pnpm typecheck`
