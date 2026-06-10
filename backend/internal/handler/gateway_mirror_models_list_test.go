package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestGatewayModels_MirrorGroupReturnsClientFacingMappingModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceGroupID := int64(30)
	mirrorGroupID := int64(31)
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
		byGroup: map[int64][]service.Account{
			sourceGroupID: {
				{
					ID:       1,
					Platform: service.PlatformOpenAI,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"gpt-5.4": "gpt-5.4-internal",
						},
					},
				},
			},
			mirrorGroupID: {
				{
					ID:       2,
					Platform: service.PlatformAnthropic,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"claude-sonnet-4-6": "claude-sonnet-4-6",
						},
					},
				},
			},
		},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:                   mirrorGroupID,
			Platform:             service.PlatformAnthropic,
			MirrorSourceGroupID:  &sourceGroupID,
			MirrorSourcePlatform: service.PlatformOpenAI,
			MirrorModelMapping: map[string]string{
				"qwen3.6-plus": "gpt-5.4",
			},
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"qwen3.6-plus"}, modelIDsForTest(got.Data))
}

func TestGatewayModels_MirrorGroupWithoutMappingUsesSourceGroupModels(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sourceGroupID := int64(32)
	mirrorGroupID := int64(33)
	h := newGatewayModelsHandlerForTest(&gatewayModelsAccountRepoStub{
		byGroup: map[int64][]service.Account{
			sourceGroupID: {
				{
					ID:       1,
					Platform: service.PlatformOpenAI,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"gpt-5.4":        "gpt-5.4-internal",
							"qwen3.6-plus":   "qwen3.6-plus",
							"z-custom-model": "z-custom-model",
						},
					},
				},
			},
			mirrorGroupID: {
				{
					ID:       2,
					Platform: service.PlatformAnthropic,
					Credentials: map[string]any{
						"model_mapping": map[string]any{
							"claude-sonnet-4-6": "claude-sonnet-4-6",
						},
					},
				},
			},
		},
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		Group: &service.Group{
			ID:                   mirrorGroupID,
			Platform:             service.PlatformAnthropic,
			MirrorSourceGroupID:  &sourceGroupID,
			MirrorSourcePlatform: service.PlatformOpenAI,
		},
	})

	h.Models(c)

	require.Equal(t, http.StatusOK, rec.Code)

	var got gatewayModelsResponseForTest
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, []string{"gpt-5.4", "qwen3.6-plus", "z-custom-model"}, modelIDsForTest(got.Data))
}

func TestMirrorRouting_AnthropicMirrorToOpenAIUsesSourceModel(t *testing.T) {
	sourceGroupID := int64(40)
	mirrorGroupID := int64(41)
	apiKey := &service.APIKey{
		GroupID: &mirrorGroupID,
		Group: &service.Group{
			ID:                   mirrorGroupID,
			Platform:             service.PlatformAnthropic,
			MirrorSourceGroupID:  &sourceGroupID,
			MirrorSourcePlatform: service.PlatformOpenAI,
			MirrorModelMapping: map[string]string{
				"qwen3.6-plus": "gpt-5.4",
			},
		},
	}
	body := []byte(`{"model":"qwen3.6-plus","messages":[]}`)

	updated, effectiveModel, mapped, err := applyMirrorModelMappingToBody(
		body,
		apiKey,
		"qwen3.6-plus",
	)

	require.NoError(t, err)
	require.True(t, mapped)
	require.Equal(t, "gpt-5.4", effectiveModel)
	require.Equal(t, "gpt-5.4", gjson.GetBytes(updated, "model").String())
	require.Equal(t, &sourceGroupID, service.APIKeyRoutingGroupID(apiKey))
	require.Equal(t, service.PlatformOpenAI, service.APIKeyRoutingPlatform(apiKey))
}
