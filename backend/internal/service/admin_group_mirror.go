package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *adminServiceImpl) SetGroupMirror(ctx context.Context, id int64, input SetGroupMirrorInput) (*Group, error) {
	repo, err := requireGroupMirrorRepository(s.groupRepo)
	if err != nil {
		return nil, err
	}
	source, err := s.groupRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	targetPlatform := strings.TrimSpace(input.TargetPlatform)
	if !isSupportedGroupMirrorPair(source.Platform, targetPlatform) {
		return nil, infraerrors.BadRequest("UNSUPPORTED_GROUP_MIRROR", "only openai <-> anthropic group mirrors are supported")
	}
	if source.IsMirror() {
		return nil, infraerrors.BadRequest("MIRROR_SOURCE_INVALID", "mirror groups cannot create mirror groups")
	}
	if input.Enabled {
		return s.enableGroupMirror(ctx, repo, source, targetPlatform, input.MirrorModelMapping)
	}
	return s.disableGroupMirror(ctx, repo, source, targetPlatform)
}

func (s *adminServiceImpl) enableGroupMirror(ctx context.Context, repo GroupMirrorRepository, source *Group, targetPlatform string, mapping map[string]string) (*Group, error) {
	existing, err := repo.GetMirrorBySourceAndPlatform(ctx, source.ID, targetPlatform)
	if err == nil {
		existing.MirrorModelMapping = normalizeGroupMirrorModelMapping(mapping)
		if err := repo.UpdateMirrorModelMapping(ctx, existing.ID, existing.MirrorModelMapping); err != nil {
			return nil, err
		}
		s.invalidateGroupAuthCaches(ctx, existing.ID)
		return s.groupRepo.GetByID(ctx, existing.ID)
	}
	if !errors.Is(err, ErrGroupNotFound) {
		return nil, err
	}
	mirror := buildMirrorGroup(source, targetPlatform, mapping)
	if err := s.groupRepo.Create(ctx, mirror); err != nil {
		return nil, err
	}
	return mirror, nil
}

func (s *adminServiceImpl) disableGroupMirror(ctx context.Context, repo GroupMirrorRepository, source *Group, targetPlatform string) (*Group, error) {
	mirror, err := repo.GetMirrorBySourceAndPlatform(ctx, source.ID, targetPlatform)
	if err != nil {
		return nil, err
	}
	if err := s.DeleteGroup(ctx, mirror.ID); err != nil {
		return nil, err
	}
	return source, nil
}

func (s *adminServiceImpl) updateMirrorGroup(ctx context.Context, group *Group, input *UpdateGroupInput) (*Group, error) {
	if input == nil || input.MirrorModelMapping == nil {
		return nil, infraerrors.BadRequest("MIRROR_MODEL_MAPPING_REQUIRED", "mirror groups can only update mirror_model_mapping")
	}
	if mirrorUpdateHasForbiddenFields(input) {
		return nil, infraerrors.BadRequest("MIRROR_GROUP_LOCKED", "mirror groups can only update mirror_model_mapping")
	}
	repo, err := requireGroupMirrorRepository(s.groupRepo)
	if err != nil {
		return nil, err
	}
	group.MirrorModelMapping = normalizeGroupMirrorModelMapping(*input.MirrorModelMapping)
	if err := repo.UpdateMirrorModelMapping(ctx, group.ID, group.MirrorModelMapping); err != nil {
		return nil, err
	}
	s.invalidateGroupAuthCaches(ctx, group.ID)
	return s.groupRepo.GetByID(ctx, group.ID)
}

func (s *adminServiceImpl) deleteGroupMirrors(ctx context.Context, source *Group) error {
	if source == nil || source.IsMirror() {
		return nil
	}
	repo, ok := s.groupRepo.(GroupMirrorRepository)
	if !ok {
		return nil
	}
	mirrors, err := repo.ListMirrorsBySourceID(ctx, source.ID)
	if err != nil {
		return err
	}
	for i := range mirrors {
		if err := s.DeleteGroup(ctx, mirrors[i].ID); err != nil {
			return err
		}
	}
	return nil
}

func requireGroupMirrorRepository(repo GroupRepository) (GroupMirrorRepository, error) {
	out, ok := repo.(GroupMirrorRepository)
	if !ok {
		return nil, infraerrors.InternalServer("GROUP_MIRROR_REPOSITORY_UNAVAILABLE", "group mirror repository is not configured")
	}
	return out, nil
}

func buildMirrorGroup(source *Group, targetPlatform string, mapping map[string]string) *Group {
	mirror := *source
	mirror.ID = 0
	mirror.Name = mirrorGroupName(source, targetPlatform)
	mirror.Description = fmt.Sprintf("Mirror of %s for %s clients", source.Name, targetPlatform)
	mirror.Platform = targetPlatform
	mirror.MirrorSourceGroupID = &source.ID
	mirror.MirrorSourcePlatform = source.Platform
	mirror.MirrorModelMapping = normalizeGroupMirrorModelMapping(mapping)
	mirror.AccountGroups = nil
	mirror.AccountCount = 0
	mirror.ActiveAccountCount = 0
	mirror.RateLimitedAccountCount = 0
	return &mirror
}

func mirrorGroupName(source *Group, targetPlatform string) string {
	return fmt.Sprintf("%s [%s mirror]", strings.TrimSpace(source.Name), targetPlatform)
}

func mirrorUpdateHasForbiddenFields(input *UpdateGroupInput) bool {
	return input.Name != "" || input.Description != "" || input.Platform != "" ||
		input.RateMultiplier != nil || input.IsExclusive != nil || input.Status != "" ||
		input.SubscriptionType != "" || input.DailyLimitUSD != nil || input.WeeklyLimitUSD != nil ||
		input.MonthlyLimitUSD != nil || input.AllowImageGeneration != nil || input.ImageRateIndependent != nil ||
		input.ImageRateMultiplier != nil || input.ImagePrice1K != nil || input.ImagePrice2K != nil ||
		input.ImagePrice4K != nil || input.ClaudeCodeOnly != nil || input.FallbackGroupID != nil ||
		input.FallbackGroupIDOnInvalidRequest != nil || input.ModelRouting != nil ||
		input.ModelRoutingEnabled != nil || input.MCPXMLInject != nil || input.SupportedModelScopes != nil ||
		input.AllowMessagesDispatch != nil || input.AllowCrossPlatformFallback != nil ||
		input.DefaultMappedModel != nil || input.RequireOAuthOnly != nil || input.RequirePrivacySet != nil ||
		input.MessagesDispatchModelConfig != nil || input.ModelsListConfig != nil ||
		input.RPMLimit != nil || input.KiroCacheEmulationEnabled != nil ||
		input.KiroCacheEmulationRatio != nil || len(input.CopyAccountsFromGroupIDs) > 0
}

func (s *adminServiceImpl) invalidateGroupAuthCaches(ctx context.Context, groupID int64) {
	if s.authCacheInvalidator == nil {
		return
	}
	s.authCacheInvalidator.InvalidateAuthCacheByGroupID(ctx, groupID)
}
