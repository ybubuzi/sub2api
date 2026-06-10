package handler

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) writeMirrorModelsList(c *gin.Context, apiKey *service.APIKey, platform string) bool {
	group := mirrorModelsListGroup(apiKey)
	if group == nil {
		return false
	}
	models := group.MirrorClientModelIDs()
	if len(models) == 0 {
		models = h.mirrorSourceModels(c.Request.Context(), apiKey)
	}
	if len(models) == 0 {
		models = defaultModelIDsForPlatform(group.EffectiveRoutingPlatform())
	}
	writeCustomModelsList(c, mirrorModelsListPlatform(group, platform), models)
	return true
}

func mirrorModelsListGroup(apiKey *service.APIKey) *service.Group {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.IsMirror() {
		return nil
	}
	return apiKey.Group
}

func (h *GatewayHandler) mirrorSourceModels(ctx context.Context, apiKey *service.APIKey) []string {
	if h == nil || h.gatewayService == nil {
		return nil
	}
	group := mirrorModelsListGroup(apiKey)
	if group == nil {
		return nil
	}
	models := h.gatewayService.GetAvailableModels(
		ctx,
		service.APIKeyRoutingGroupID(apiKey),
		group.EffectiveRoutingPlatform(),
	)
	if group.CustomModelsListEnabled() {
		return filterModelsByCustomList(
			models,
			defaultModelIDsForPlatform(group.EffectiveRoutingPlatform()),
			group.ModelsListConfig.Models,
		)
	}
	return models
}

func mirrorModelsListPlatform(group *service.Group, platform string) string {
	platform = strings.TrimSpace(platform)
	if platform != "" {
		return platform
	}
	return group.Platform
}
