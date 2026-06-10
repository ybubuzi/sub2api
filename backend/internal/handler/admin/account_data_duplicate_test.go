package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestImportDataOverwritesExistingAccountByNameByDefault(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = []service.Account{existingDuplicateAccount()}

	rec := postDuplicateAccountImport(t, router, "")
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeImportResponse(t, rec)
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 0, resp.Data.AccountCreated)
	require.Equal(t, 1, resp.Data.AccountUpdated)
	require.Equal(t, 0, resp.Data.AccountIgnored)
	require.Equal(t, 0, resp.Data.AccountFailed)
	require.Empty(t, adminSvc.createdAccounts)
	require.Len(t, adminSvc.updatedAccounts, 1)
	require.Equal(t, int64(41), adminSvc.updatedAccountIDs[0])
	require.Equal(t, "imported", adminSvc.updatedAccounts[0].Credentials["token"])
	require.NotEmpty(t, adminSvc.updatedAccounts[0].Extra["imported_at"])
}

func TestImportDataIgnoresExistingAccountByName(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = []service.Account{existingDuplicateAccount()}

	rec := postDuplicateAccountImport(t, router, duplicateAccountIgnore)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeImportResponse(t, rec)
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 0, resp.Data.AccountCreated)
	require.Equal(t, 0, resp.Data.AccountUpdated)
	require.Equal(t, 1, resp.Data.AccountIgnored)
	require.Equal(t, 0, resp.Data.AccountFailed)
	require.Empty(t, adminSvc.createdAccounts)
	require.Empty(t, adminSvc.updatedAccounts)
}

func TestImportDataCopiesExistingAccountByName(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = []service.Account{existingDuplicateAccount()}

	rec := postDuplicateAccountImport(t, router, duplicateAccountCopy)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeImportResponse(t, rec)
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, resp.Data.AccountCreated)
	require.Equal(t, 0, resp.Data.AccountUpdated)
	require.Equal(t, 0, resp.Data.AccountIgnored)
	require.Equal(t, 0, resp.Data.AccountFailed)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.Empty(t, adminSvc.updatedAccounts)
	require.Equal(t, "duplicate@example.com", adminSvc.createdAccounts[0].Name)
	require.NotEmpty(t, adminSvc.createdAccounts[0].Extra["imported_at"])
}

func TestImportDataCreatedAccountRecordsImportedAt(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = nil

	rec := postDuplicateAccountImport(t, router, "")
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeImportResponse(t, rec)
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, resp.Data.AccountCreated)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.NotEmpty(t, adminSvc.createdAccounts[0].Extra["imported_at"])
}

func TestImportDataRejectsInvalidDuplicateAccountAction(t *testing.T) {
	router, _ := setupAccountDataRouter()

	rec := postDuplicateAccountImport(t, router, "replace")

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestImportDataAppliesPlatformGroupsAndBatchProxy(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.accounts = nil
	adminSvc.proxies = []service.Proxy{{ID: 72, Name: "proxy", Protocol: "http", Host: "127.0.0.1", Port: 8080}}
	adminSvc.groups = []service.Group{
		{ID: 11, Name: "openai-a", Platform: service.PlatformOpenAI, Status: service.StatusActive},
		{ID: 22, Name: "claude-b", Platform: service.PlatformAnthropic, Status: service.StatusActive},
	}

	payload := multiPlatformImportPayload(map[string]any{
		"proxy_id": int64(72),
		"platform_group_ids": map[string]any{
			service.PlatformOpenAI: []int64{11},
			"claude":               []int64{22},
		},
	})
	rec := postImportPayload(t, router, payload)
	require.Equal(t, http.StatusOK, rec.Code)

	resp := decodeImportResponse(t, rec)
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 2, resp.Data.AccountCreated)
	require.Equal(t, 0, resp.Data.AccountFailed)
	require.Len(t, adminSvc.createdAccounts, 2)
	require.Equal(t, int64(72), *adminSvc.createdAccounts[0].ProxyID)
	require.Equal(t, []int64{11}, adminSvc.createdAccounts[0].GroupIDs)
	require.Equal(t, int64(72), *adminSvc.createdAccounts[1].ProxyID)
	require.Equal(t, []int64{22}, adminSvc.createdAccounts[1].GroupIDs)
}

func TestImportDataRejectsPlatformGroupMismatch(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()
	adminSvc.groups = []service.Group{
		{ID: 11, Name: "wrong", Platform: service.PlatformAnthropic, Status: service.StatusActive},
	}

	payload := duplicateAccountImportPayload("")
	payload["platform_group_ids"] = map[string]any{service.PlatformOpenAI: []int64{11}}
	rec := postImportPayload(t, router, payload)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func existingDuplicateAccount() service.Account {
	return service.Account{
		ID:          41,
		Name:        "duplicate@example.com",
		Platform:    service.PlatformOpenAI,
		Type:        service.AccountTypeOAuth,
		Credentials: map[string]any{"token": "existing"},
		Status:      service.StatusActive,
	}
}

func postDuplicateAccountImport(t *testing.T, handler http.Handler, action string) *httptest.ResponseRecorder {
	t.Helper()
	payload := duplicateAccountImportPayload(action)
	return postImportPayload(t, handler, payload)
}

func postImportPayload(t *testing.T, handler http.Handler, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(rec, req)
	return rec
}

func duplicateAccountImportPayload(action string) map[string]any {
	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{
				{
					"name":        "duplicate@example.com",
					"platform":    service.PlatformOpenAI,
					"type":        service.AccountTypeOAuth,
					"credentials": map[string]any{"token": "imported"},
					"extra":       map[string]any{"source": "test"},
					"concurrency": 3,
					"priority":    50,
				},
			},
		},
		"skip_default_group_bind": true,
	}
	if action != "" {
		payload["duplicate_account_action"] = action
	}
	return payload
}

func multiPlatformImportPayload(extra map[string]any) map[string]any {
	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{},
			"accounts": []map[string]any{
				{
					"name":        "openai@example.com",
					"platform":    service.PlatformOpenAI,
					"type":        service.AccountTypeOAuth,
					"credentials": map[string]any{"token": "openai"},
					"concurrency": 3,
					"priority":    50,
				},
				{
					"name":        "claude@example.com",
					"platform":    service.PlatformAnthropic,
					"type":        service.AccountTypeOAuth,
					"credentials": map[string]any{"token": "claude"},
					"concurrency": 3,
					"priority":    50,
				},
			},
		},
		"skip_default_group_bind": true,
	}
	for key, value := range extra {
		payload[key] = value
	}
	return payload
}

func decodeImportResponse(t *testing.T, rec *httptest.ResponseRecorder) dataImportResponse {
	t.Helper()
	var resp dataImportResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp
}
