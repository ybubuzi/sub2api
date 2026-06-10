-- OpenAI/Claude mirror groups.
-- mirror_source_group_id points to the original group; the mirror group's own
-- platform remains the client-facing platform shown in admin/user selectors.

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS mirror_source_group_id BIGINT,
  ADD COLUMN IF NOT EXISTS mirror_source_platform VARCHAR(50) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS mirror_model_mapping JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_groups_mirror_source_group_id
  ON groups (mirror_source_group_id)
  WHERE mirror_source_group_id IS NOT NULL AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_mirror_source_platform_active
  ON groups (mirror_source_group_id, platform)
  WHERE mirror_source_group_id IS NOT NULL AND deleted_at IS NULL;

COMMENT ON COLUMN groups.mirror_source_group_id IS 'Original group id for an OpenAI/Claude mirror group.';
COMMENT ON COLUMN groups.mirror_source_platform IS 'Original group platform captured when the mirror is created.';
COMMENT ON COLUMN groups.mirror_model_mapping IS 'Mirror-only request model to upstream model mapping.';
