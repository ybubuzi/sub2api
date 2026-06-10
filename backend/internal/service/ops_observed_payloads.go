package service

import (
	"encoding/json"
	"regexp"
	"strings"
)

const opsMaxStoredObservedBodyBytes = 64 * 1024

// SanitizeOpsObservedHeaders preserves header names and values for debugging,
// while redacting credential-like values before they reach persistent storage.
func SanitizeOpsObservedHeaders(headers map[string][]string) (*string, error) {
	if len(headers) == 0 {
		return nil, nil
	}

	out := make(map[string][]string, len(headers))
	for key, values := range headers {
		name := strings.TrimSpace(key)
		if name == "" {
			continue
		}
		out[name] = sanitizeOpsObservedHeaderValues(name, values)
	}
	if len(out) == 0 {
		return nil, nil
	}

	encoded, err := json.Marshal(out)
	if err != nil {
		return nil, err
	}
	s := string(encoded)
	return &s, nil
}

func sanitizeOpsObservedHeaderValues(key string, values []string) []string {
	out := make([]string, len(values))
	if isSensitiveKey(key) {
		for i := range out {
			out[i] = "[REDACTED]"
		}
		return out
	}
	copy(out, values)
	return out
}

func prepareOpsObservedPayloads(entry *OpsInsertErrorLogInput) {
	if entry == nil {
		return
	}
	entry.ObservedRequestBody, entry.ObservedRequestBodyTruncated = sanitizeOpsObservedBody(
		entry.ObservedRequestBody,
		entry.ObservedRequestBodyTruncated,
	)
	entry.ObservedResponseBody, entry.ObservedResponseBodyTruncated = sanitizeOpsObservedBody(
		entry.ObservedResponseBody,
		entry.ObservedResponseBodyTruncated,
	)
	setObservedBodyBytes(&entry.ObservedRequestBodyBytes, entry.ObservedRequestBody, entry.ObservedRequestBodyTruncated)
	setObservedBodyBytes(&entry.ObservedResponseBodyBytes, entry.ObservedResponseBody, entry.ObservedResponseBodyTruncated)
}

func sanitizeOpsObservedBody(raw string, wasTruncated bool) (string, bool) {
	if strings.TrimSpace(raw) == "" {
		return "", wasTruncated
	}
	sanitized, truncated := sanitizeErrorBodyForStorage(raw, opsMaxStoredObservedBodyBytes)
	return redactSensitiveObservedText(sanitized), wasTruncated || truncated
}

func setObservedBodyBytes(target **int, body string, truncated bool) {
	if target == nil || *target != nil || body == "" || truncated {
		return
	}
	v := len(body)
	*target = &v
}

var opsObservedSensitiveTextPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)("(?:authorization|proxy-authorization|x-api-key|api_key|apikey|access_token|refresh_token|id_token|session_token|token|password|passwd|passphrase|secret|client_secret|private_key|jwt|signature)"\s*:\s*")[^"]*(")`),
	regexp.MustCompile(`(?i)((?:authorization|proxy-authorization|x-api-key|api_key|apikey|access_token|refresh_token|id_token|session_token|token|password|passwd|passphrase|secret|client_secret|private_key|jwt|signature)=)[^&\s]+`),
}

func redactSensitiveObservedText(raw string) string {
	out := raw
	for _, pattern := range opsObservedSensitiveTextPatterns {
		out = pattern.ReplaceAllString(out, `${1}[REDACTED]${2}`)
	}
	return out
}
