package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

func (r *groupRepository) updateGroupMirrorFields(ctx context.Context, groupIn *service.Group) error {
	if r == nil || r.sql == nil || groupIn == nil || groupIn.ID <= 0 {
		return nil
	}
	mapping, err := json.Marshal(groupIn.MirrorModelMapping)
	if err != nil {
		return err
	}
	_, err = r.sql.ExecContext(ctx, `
		UPDATE groups
		SET mirror_source_group_id = $1,
			mirror_source_platform = $2,
			mirror_model_mapping = COALESCE($3::jsonb, '{}'::jsonb)
		WHERE id = $4`,
		groupIn.MirrorSourceGroupID,
		strings.TrimSpace(groupIn.MirrorSourcePlatform),
		string(mapping),
		groupIn.ID,
	)
	return err
}

func (r *groupRepository) hydrateGroupMirrorFields(ctx context.Context, groupIn *service.Group) error {
	if groupIn == nil {
		return nil
	}
	groups := []service.Group{*groupIn}
	if err := r.hydrateGroupMirrorFieldsForGroups(ctx, groups); err != nil {
		return err
	}
	*groupIn = groups[0]
	return nil
}

func (r *groupRepository) hydrateGroupMirrorFieldsForGroups(ctx context.Context, groups []service.Group) error {
	if r == nil || r.sql == nil || len(groups) == 0 {
		return nil
	}
	ids, indexByID := groupIDsWithIndex(groups)
	if len(ids) == 0 {
		return nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id, mirror_source_group_id, mirror_source_platform, mirror_model_mapping
		FROM groups
		WHERE id = ANY($1)`, pq.Array(ids))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		if err := scanGroupMirrorRow(rows, groups, indexByID); err != nil {
			return err
		}
	}
	return rows.Err()
}

func (r *groupRepository) GetMirrorBySourceAndPlatform(ctx context.Context, sourceID int64, platform string) (*service.Group, error) {
	if sourceID <= 0 || strings.TrimSpace(platform) == "" {
		return nil, service.ErrGroupNotFound
	}
	return r.getGroupByMirrorQuery(ctx, `
		SELECT id FROM groups
		WHERE mirror_source_group_id = $1 AND platform = $2 AND deleted_at IS NULL
		LIMIT 1`, sourceID, strings.TrimSpace(platform))
}

func (r *groupRepository) ListMirrorsBySourceID(ctx context.Context, sourceID int64) ([]service.Group, error) {
	if sourceID <= 0 {
		return nil, nil
	}
	rows, err := r.sql.QueryContext(ctx, `
		SELECT id FROM groups
		WHERE mirror_source_group_id = $1 AND deleted_at IS NULL
		ORDER BY id`, sourceID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []service.Group
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		groupOut, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *groupOut)
	}
	return out, rows.Err()
}

func (r *groupRepository) UpdateMirrorModelMapping(ctx context.Context, groupID int64, mapping map[string]string) error {
	if groupID <= 0 {
		return service.ErrGroupNotFound
	}
	normalized := service.NormalizeGroupMirrorModelMappingForRepository(mapping)
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	result, err := r.sql.ExecContext(ctx, `
		UPDATE groups
		SET mirror_model_mapping = COALESCE($1::jsonb, '{}'::jsonb), updated_at = NOW()
		WHERE id = $2 AND mirror_source_group_id IS NOT NULL AND deleted_at IS NULL`,
		string(encoded), groupID)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrGroupNotFound
	}
	return nil
}

func (r *groupRepository) getGroupByMirrorQuery(ctx context.Context, query string, args ...any) (*service.Group, error) {
	rows, err := r.sql.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var id int64
	if rows.Next() {
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if id == 0 {
		return nil, service.ErrGroupNotFound
	}
	return r.GetByID(ctx, id)
}

func groupIDsWithIndex(groups []service.Group) ([]int64, map[int64]int) {
	ids := make([]int64, 0, len(groups))
	indexByID := make(map[int64]int, len(groups))
	for i := range groups {
		if groups[i].ID <= 0 {
			continue
		}
		ids = append(ids, groups[i].ID)
		indexByID[groups[i].ID] = i
	}
	return ids, indexByID
}

func scanGroupMirrorRow(rows *sql.Rows, groups []service.Group, indexByID map[int64]int) error {
	var id int64
	var sourceID sql.NullInt64
	var sourcePlatform string
	var mappingBytes []byte
	if err := rows.Scan(&id, &sourceID, &sourcePlatform, &mappingBytes); err != nil {
		return err
	}
	idx, ok := indexByID[id]
	if !ok {
		return nil
	}
	return applyGroupMirrorRow(&groups[idx], sourceID, sourcePlatform, mappingBytes)
}

func applyGroupMirrorRow(groupIn *service.Group, sourceID sql.NullInt64, sourcePlatform string, mappingBytes []byte) error {
	groupIn.MirrorSourceGroupID = nil
	if sourceID.Valid && sourceID.Int64 > 0 {
		id := sourceID.Int64
		groupIn.MirrorSourceGroupID = &id
	}
	groupIn.MirrorSourcePlatform = strings.TrimSpace(sourcePlatform)
	groupIn.MirrorModelMapping = nil
	if len(mappingBytes) == 0 {
		return nil
	}
	var mapping map[string]string
	if err := json.Unmarshal(mappingBytes, &mapping); err != nil {
		return fmt.Errorf("decode group mirror model mapping: %w", err)
	}
	groupIn.MirrorModelMapping = service.NormalizeGroupMirrorModelMappingForRepository(mapping)
	return nil
}
