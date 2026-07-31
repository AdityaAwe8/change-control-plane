package types

import (
	"strconv"
	"strings"
)

const RedactedMetadataValue = "[redacted]"

func RedactMetadata(metadata Metadata) Metadata {
	if metadata == nil {
		return nil
	}
	redacted := Metadata{}
	for key, value := range metadata {
		redacted[key] = redactMetadataValueForKey(key, value)
	}
	return redacted
}

func RedactMetadataValue(value any) any {
	return redactMetadataValueForKey("", value)
}

func FirstUnsafeMetadataPath(value any, path string) string {
	switch typed := value.(type) {
	case Metadata:
		for key, child := range typed {
			childPath := joinMetadataPath(path, key)
			if SensitiveMetadataKey(key) && !SafeMetadataReferenceKey(key) {
				return childPath
			}
			if MetadataStringLooksSensitive(key, child) {
				return childPath
			}
			if nestedPath := FirstUnsafeMetadataPath(child, childPath); nestedPath != "" {
				return nestedPath
			}
		}
	case map[string]any:
		return FirstUnsafeMetadataPath(Metadata(typed), path)
	case map[string]string:
		for key, child := range typed {
			childPath := joinMetadataPath(path, key)
			if SensitiveMetadataKey(key) && !SafeMetadataReferenceKey(key) {
				return childPath
			}
			if MetadataStringLooksSensitive(key, child) {
				return childPath
			}
		}
	case []any:
		for idx, child := range typed {
			if nestedPath := FirstUnsafeMetadataPath(child, path+"["+strconv.Itoa(idx)+"]"); nestedPath != "" {
				return nestedPath
			}
			if MetadataStringLooksSensitive("", child) {
				return path + "[" + strconv.Itoa(idx) + "]"
			}
		}
	case []string:
		for idx, child := range typed {
			if MetadataStringLooksSensitive("", child) {
				return path + "[" + strconv.Itoa(idx) + "]"
			}
		}
	}
	return ""
}

func SensitiveMetadataKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	compact := compactMetadataKey(normalized)
	switch compact {
	case "token", "accesstoken", "apikey", "authorization", "bearer", "bearertoken", "clientsecret", "dsn", "password", "privatekey", "secret", "webhooksecret", "xapikey":
		return true
	}
	for _, suffix := range []string{"token", "secret", "password", "privatekey", "apikey", "authorization", "bearer", "dsn"} {
		if strings.HasSuffix(compact, suffix) {
			return true
		}
	}
	return false
}

func SafeMetadataReferenceKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	compact := compactMetadataKey(normalized)
	return strings.HasSuffix(normalized, "_env") ||
		strings.HasSuffix(normalized, "_env_var") ||
		strings.HasSuffix(normalized, "_secret_ref") ||
		normalized == "secret_ref" ||
		normalized == "secret_ref_env" ||
		normalized == "dsn_env" ||
		strings.HasSuffix(compact, "env") ||
		strings.HasSuffix(compact, "envvar") ||
		strings.HasSuffix(compact, "secretref") ||
		compact == "secretref" ||
		compact == "secretrefenv" ||
		compact == "dsnenv"
}

func MetadataStringLooksSensitive(key string, value any) bool {
	raw, ok := value.(string)
	if !ok {
		return false
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return false
	}
	lower := strings.ToLower(trimmed)
	if strings.Contains(lower, "-----begin ") && strings.Contains(lower, "private key") {
		return true
	}
	if strings.HasPrefix(lower, "bearer ") || strings.HasPrefix(lower, "basic ") {
		return true
	}
	if strings.Contains(lower, "password=") {
		return true
	}
	if looksLikeDSN(lower) {
		return true
	}
	if looksLikeProviderToken(trimmed) {
		return true
	}
	if SafeMetadataReferenceKey(key) {
		return false
	}
	return SensitiveMetadataKey(key)
}

func redactMetadataValueForKey(key string, value any) any {
	if SensitiveMetadataKey(key) && !SafeMetadataReferenceKey(key) {
		return RedactedMetadataValue
	}
	if MetadataStringLooksSensitive(key, value) {
		return RedactedMetadataValue
	}
	switch typed := value.(type) {
	case Metadata:
		return RedactMetadata(typed)
	case map[string]any:
		return RedactMetadata(Metadata(typed))
	case map[string]string:
		redacted := map[string]any{}
		for childKey, childValue := range typed {
			redacted[childKey] = redactMetadataValueForKey(childKey, childValue)
		}
		return redacted
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, redactMetadataValueForKey("", item))
		}
		return items
	case []string:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, redactMetadataValueForKey("", item))
		}
		return items
	default:
		return value
	}
}

func compactMetadataKey(key string) string {
	return strings.NewReplacer("_", "", "-", "", ".", "").Replace(key)
}

func looksLikeDSN(lower string) bool {
	for _, prefix := range []string{
		"postgres://",
		"postgresql://",
		"mysql://",
		"mariadb://",
		"sqlserver://",
		"mongodb://",
		"redis://",
	} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func looksLikeProviderToken(value string) bool {
	for _, prefix := range []string{
		"ccpt_",
		"ghp_",
		"github_pat_",
		"glpat-",
		"sk-",
		"xoxb-",
		"xoxp-",
		"ya29.",
		"eyJ",
	} {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func joinMetadataPath(path, key string) string {
	if strings.TrimSpace(path) == "" {
		return key
	}
	return path + "." + key
}
