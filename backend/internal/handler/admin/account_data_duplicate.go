package admin

import (
	"context"
	"fmt"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type dataImportOptions struct {
	duplicateAction  string
	batchProxyID     *int64
	platformGroupIDs map[string][]int64
}

func normalizeDuplicateAccountAction(action string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(action))
	if normalized == "" {
		return duplicateAccountOverwrite, nil
	}
	switch normalized {
	case duplicateAccountOverwrite, duplicateAccountCopy, duplicateAccountIgnore:
		return normalized, nil
	default:
		return "", fmt.Errorf("invalid duplicate_account_action: %s", action)
	}
}

func cloneImportMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}

func (h *AccountHandler) resolveDataImportOptions(ctx context.Context, req DataImportRequest, proxies []service.Proxy) (dataImportOptions, error) {
	duplicateAction, err := normalizeDuplicateAccountAction(req.DuplicateAccountAction)
	if err != nil {
		return dataImportOptions{}, err
	}
	batchProxyID, err := resolveBatchProxyID(req.ProxyID, proxies)
	if err != nil {
		return dataImportOptions{}, err
	}
	platformGroupIDs, err := h.resolveImportPlatformGroupIDs(ctx, req.PlatformGroupIDs)
	if err != nil {
		return dataImportOptions{}, err
	}
	return dataImportOptions{duplicateAction: duplicateAction, batchProxyID: batchProxyID, platformGroupIDs: platformGroupIDs}, nil
}

func (opts dataImportOptions) groupIDsForPlatform(platform string) ([]int64, bool) {
	normalized, err := normalizeImportPlatform(platform)
	if err != nil {
		return nil, false
	}
	groupIDs, ok := opts.platformGroupIDs[normalized]
	return append([]int64(nil), groupIDs...), ok
}

func (h *AccountHandler) createImportedAccount(ctx context.Context, item DataAccount, proxyID *int64, groupIDs []int64, skipDefaultGroupBind bool, importedAt string) (*service.Account, error) {
	extra := cloneImportMap(item.Extra)
	extra["imported_at"] = importedAt
	input := &service.CreateAccountInput{
		Name:                 item.Name,
		Notes:                item.Notes,
		Platform:             item.Platform,
		Type:                 item.Type,
		Credentials:          item.Credentials,
		Extra:                extra,
		ProxyID:              proxyID,
		Concurrency:          item.Concurrency,
		Priority:             item.Priority,
		RateMultiplier:       item.RateMultiplier,
		GroupIDs:             append([]int64(nil), groupIDs...),
		ExpiresAt:            item.ExpiresAt,
		AutoPauseOnExpired:   item.AutoPauseOnExpired,
		SkipDefaultGroupBind: skipDefaultGroupBind,
	}
	return h.adminService.CreateAccount(ctx, input)
}

func (h *AccountHandler) overwriteImportedAccount(ctx context.Context, accountID int64, item DataAccount, proxyID *int64, groupIDs *[]int64, importedAt string) error {
	extra := cloneImportMap(item.Extra)
	extra["imported_at"] = importedAt
	_, err := h.adminService.UpdateAccount(ctx, accountID, &service.UpdateAccountInput{
		Name:               item.Name,
		Notes:              item.Notes,
		Type:               item.Type,
		Credentials:        item.Credentials,
		Extra:              extra,
		ProxyID:            proxyID,
		Concurrency:        &item.Concurrency,
		Priority:           &item.Priority,
		GroupIDs:           groupIDs,
		RateMultiplier:     item.RateMultiplier,
		ExpiresAt:          item.ExpiresAt,
		AutoPauseOnExpired: item.AutoPauseOnExpired,
	})
	return err
}

func buildAccountByName(accounts []service.Account) map[string]service.Account {
	out := make(map[string]service.Account, len(accounts))
	for i := range accounts {
		name := normalizeImportAccountName(accounts[i].Name)
		if name == "" {
			continue
		}
		if _, exists := out[name]; exists {
			continue
		}
		out[name] = accounts[i]
	}
	return out
}

func normalizeImportAccountName(name string) string {
	return strings.TrimSpace(name)
}

func resolveBatchProxyID(proxyID *int64, proxies []service.Proxy) (*int64, error) {
	if proxyID == nil {
		return nil, nil
	}
	if *proxyID < 0 {
		return nil, badDataImportOptions("proxy_id must be >= 0")
	}
	if *proxyID == 0 {
		cleared := int64(0)
		return &cleared, nil
	}
	for i := range proxies {
		if proxies[i].ID == *proxyID {
			id := *proxyID
			return &id, nil
		}
	}
	return nil, badDataImportOptions(fmt.Sprintf("proxy_id not found: %d", *proxyID))
}

func (h *AccountHandler) resolveImportPlatformGroupIDs(ctx context.Context, input map[string][]int64) (map[string][]int64, error) {
	out := make(map[string][]int64, len(input))
	for rawPlatform, groupIDs := range input {
		platform, err := normalizeImportPlatform(rawPlatform)
		if err != nil {
			return nil, err
		}
		if _, exists := out[platform]; exists {
			return nil, badDataImportOptions(fmt.Sprintf("duplicate platform group mapping: %s", rawPlatform))
		}
		resolved, err := h.resolveImportGroupIDs(ctx, platform, groupIDs)
		if err != nil {
			return nil, err
		}
		out[platform] = resolved
	}
	return out, nil
}

func (h *AccountHandler) resolveImportGroupIDs(ctx context.Context, platform string, groupIDs []int64) ([]int64, error) {
	normalized := normalizeImportGroupIDs(groupIDs)
	if len(normalized) == 0 {
		return nil, badDataImportOptions(fmt.Sprintf("platform_group_ids.%s cannot be empty", platform))
	}
	for _, groupID := range normalized {
		group, err := h.adminService.GetGroup(ctx, groupID)
		if err != nil {
			return nil, err
		}
		if group == nil {
			return nil, badDataImportOptions(fmt.Sprintf("group not found: %d", groupID))
		}
		if group.Platform != platform {
			return nil, badDataImportOptions(fmt.Sprintf("group %d platform mismatch: expected=%s actual=%s", groupID, platform, group.Platform))
		}
	}
	return normalized, nil
}

func normalizeImportGroupIDs(groupIDs []int64) []int64 {
	seen := make(map[int64]struct{}, len(groupIDs))
	out := make([]int64, 0, len(groupIDs))
	for _, id := range groupIDs {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeImportPlatform(platform string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(platform))
	if normalized == "claude" {
		return service.PlatformAnthropic, nil
	}
	switch normalized {
	case service.PlatformAnthropic, service.PlatformOpenAI, service.PlatformGemini, service.PlatformAntigravity, service.PlatformKiro:
		return normalized, nil
	default:
		return "", badDataImportOptions(fmt.Sprintf("unsupported platform: %s", platform))
	}
}

func resolveImportedAccountProxyID(item DataAccount, proxyKeyToID map[string]int64, batchProxyID *int64) (*int64, error) {
	if batchProxyID != nil {
		id := *batchProxyID
		return &id, nil
	}
	proxyKey := accountProxyKeyValue(item)
	if proxyKey == "" {
		return nil, nil
	}
	id, ok := proxyKeyToID[proxyKey]
	if !ok {
		return nil, fmt.Errorf("proxy_key not found")
	}
	return &id, nil
}

func accountProxyKeyValue(item DataAccount) string {
	if item.ProxyKey == nil {
		return ""
	}
	return strings.TrimSpace(*item.ProxyKey)
}

func groupIDsForUpdate(groupIDs []int64, hasGroupOverride bool) *[]int64 {
	if !hasGroupOverride {
		return nil
	}
	copied := append([]int64(nil), groupIDs...)
	return &copied
}

func badDataImportOptions(message string) error {
	return infraerrors.BadRequest("INVALID_DATA_IMPORT_OPTIONS", message)
}
