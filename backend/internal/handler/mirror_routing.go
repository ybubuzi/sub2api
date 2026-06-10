package handler

import (
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/tidwall/gjson"
)

func applyMirrorModelMappingToBody(body []byte, apiKey *service.APIKey, requestedModel string) ([]byte, string, bool, error) {
	if apiKey == nil || apiKey.Group == nil {
		return body, requestedModel, false, nil
	}
	mappedModel := strings.TrimSpace(apiKey.Group.ResolveMirrorMappedModel(requestedModel))
	if mappedModel == "" {
		return body, requestedModel, false, nil
	}
	updated, err := replaceRequestModelStrict(body, mappedModel)
	if err != nil {
		return nil, "", false, err
	}
	return updated, mappedModel, true, nil
}

func applyChannelModelMappingToBody(body []byte, mapping service.ChannelMappingResult) ([]byte, error) {
	if !mapping.Mapped {
		return body, nil
	}
	return replaceRequestModelStrict(body, mapping.MappedModel)
}

func replaceRequestModelStrict(body []byte, newModel string) ([]byte, error) {
	newModel = strings.TrimSpace(newModel)
	updated := service.ReplaceModelInBody(body, newModel)
	modelResult := gjson.GetBytes(updated, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String {
		return nil, fmt.Errorf("request model field missing after model replacement")
	}
	if strings.TrimSpace(modelResult.String()) != newModel {
		return nil, fmt.Errorf("request model replacement failed")
	}
	return updated, nil
}
