package service

import "strings"

const (
	GroupMirrorTargetPlatformOpenAI    = PlatformOpenAI
	GroupMirrorTargetPlatformAnthropic = PlatformAnthropic
)

type SetGroupMirrorInput struct {
	TargetPlatform     string
	Enabled            bool
	MirrorModelMapping map[string]string
}

func (g *Group) IsMirror() bool {
	return g != nil && g.MirrorSourceGroupID != nil && *g.MirrorSourceGroupID > 0
}

func (g *Group) EffectiveRoutingPlatform() string {
	if g == nil {
		return ""
	}
	if g.IsMirror() {
		return strings.TrimSpace(g.MirrorSourcePlatform)
	}
	return g.Platform
}

func (g *Group) ResolveMirrorMappedModel(requestedModel string) string {
	if g == nil || len(g.MirrorModelMapping) == 0 {
		return ""
	}
	mapped, matched := resolveRequestedModelInMapping(g.MirrorModelMapping, requestedModel)
	if !matched || strings.TrimSpace(mapped) == strings.TrimSpace(requestedModel) {
		return ""
	}
	return strings.TrimSpace(mapped)
}

func APIKeyRoutingGroupID(apiKey *APIKey) *int64 {
	if apiKey == nil {
		return nil
	}
	if apiKey.Group != nil && apiKey.Group.IsMirror() {
		id := *apiKey.Group.MirrorSourceGroupID
		return &id
	}
	return apiKey.GroupID
}

func APIKeyRoutingPlatform(apiKey *APIKey) string {
	if apiKey == nil || apiKey.Group == nil {
		return ""
	}
	return apiKey.Group.EffectiveRoutingPlatform()
}

func GroupAllowsMessagesDispatch(group *Group) bool {
	if group == nil {
		return false
	}
	if group.IsMirror() && group.EffectiveRoutingPlatform() == PlatformOpenAI {
		return true
	}
	return group.AllowMessagesDispatch
}

func normalizeGroupMirrorModelMapping(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for rawFrom, rawTo := range input {
		from := strings.TrimSpace(rawFrom)
		to := strings.TrimSpace(rawTo)
		if from == "" || to == "" {
			continue
		}
		out[from] = to
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func NormalizeGroupMirrorModelMappingForRepository(input map[string]string) map[string]string {
	return normalizeGroupMirrorModelMapping(input)
}

func isSupportedGroupMirrorPair(sourcePlatform, targetPlatform string) bool {
	switch strings.TrimSpace(sourcePlatform) {
	case PlatformOpenAI:
		return targetPlatform == PlatformAnthropic
	case PlatformAnthropic:
		return targetPlatform == PlatformOpenAI
	default:
		return false
	}
}

func sanitizeGroupMessagesDispatchFields(g *Group) {
	if g == nil || g.EffectiveRoutingPlatform() == PlatformOpenAI {
		return
	}
	g.AllowMessagesDispatch = false
	g.DefaultMappedModel = ""
	g.MessagesDispatchModelConfig = OpenAIMessagesDispatchModelConfig{}
}
