package types

import (
	"encoding/json"
	"testing"
)

func TestPublicResponseEnvelopeJSONShape(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(ItemResponse[Integration]{
		Data: Integration{
			BaseRecord:     BaseRecord{ID: "int_123"},
			OrganizationID: "org_123",
			Name:           "GitHub",
			Kind:           "github",
		},
	})
	if err != nil {
		t.Fatalf("marshal item response: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode payload map: %v", err)
	}
	if _, ok := decoded["data"]; !ok || len(decoded) != 1 {
		t.Fatalf("expected item envelope with only data, got %s", string(payload))
	}

	listPayload, err := json.Marshal(ListResponse[Integration]{Data: []Integration{{BaseRecord: BaseRecord{ID: "int_123"}}}})
	if err != nil {
		t.Fatalf("marshal list response: %v", err)
	}
	var listEnvelope struct {
		Data []Integration `json:"data"`
	}
	if err := json.Unmarshal(listPayload, &listEnvelope); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listEnvelope.Data) != 1 || listEnvelope.Data[0].ID != "int_123" {
		t.Fatalf("unexpected list envelope data: %+v", listEnvelope.Data)
	}

	emptyListPayload, err := json.Marshal(ListResponse[Integration]{})
	if err != nil {
		t.Fatalf("marshal empty list response: %v", err)
	}
	var emptyListEnvelope struct {
		Data []Integration `json:"data"`
	}
	if err := json.Unmarshal(emptyListPayload, &emptyListEnvelope); err != nil {
		t.Fatalf("decode empty list response: %v", err)
	}
	if emptyListEnvelope.Data == nil || len(emptyListEnvelope.Data) != 0 {
		t.Fatalf("expected empty list envelope data to be [], got %s", string(emptyListPayload))
	}
}

func TestPublicContractTypesPreserveNewIntegrationFields(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"organization_id":"org_123",
		"kind":"github",
		"name":"GitHub App",
		"instance_key":"github-prod",
		"scope_type":"organization",
		"scope_name":"acme",
		"mode":"advisory",
		"auth_strategy":"github_app",
		"enabled":true,
		"control_enabled":false,
		"schedule_enabled":true,
		"schedule_interval_seconds":300,
		"sync_stale_after_seconds":900,
		"metadata":{"private_key_env":"CCP_GITHUB_APP_PRIVATE_KEY"}
	}`)
	var req CreateIntegrationRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("decode create integration request: %v", err)
	}
	if req.InstanceKey != "github-prod" ||
		req.AuthStrategy != "github_app" ||
		!req.ScheduleEnabled ||
		req.ScheduleIntervalSeconds != 300 ||
		req.SyncStaleAfterSeconds != 900 ||
		req.Metadata["private_key_env"] != "CCP_GITHUB_APP_PRIVATE_KEY" {
		t.Fatalf("integration request lost public fields: %+v", req)
	}
}

func TestRedactMetadataPreservesSafeReferencesAndRedactsLegacySecrets(t *testing.T) {
	t.Parallel()

	rawDSN := "postgres" + "://db.internal:5432/app?sslmode=disable"
	metadata := Metadata{
		"access_token":   "ghp_legacy_plaintext_token",
		"secret_ref":     "prod/github/app/private-key",
		"secret_ref_env": "CCP_GITHUB_APP_SECRET_REF",
		"dsn_env":        "CCP_RUNTIME_DSN",
		"runtime_dsn":    rawDSN,
		"namespace":      "prod",
		"headers": map[string]any{
			"Authorization": "Bearer legacy-provider-token",
			"Accept":        "application/json",
		},
		"safe_refs": []any{
			Metadata{"api_key_env": "CCP_PROVIDER_API_KEY"},
			"plain-note",
		},
	}

	redacted := RedactMetadata(metadata)

	if redacted["access_token"] != RedactedMetadataValue {
		t.Fatalf("expected provider token to be redacted, got %+v", redacted["access_token"])
	}
	if redacted["runtime_dsn"] != RedactedMetadataValue {
		t.Fatalf("expected raw DSN to be redacted, got %+v", redacted["runtime_dsn"])
	}
	if redacted["secret_ref"] != "prod/github/app/private-key" || redacted["secret_ref_env"] != "CCP_GITHUB_APP_SECRET_REF" || redacted["dsn_env"] != "CCP_RUNTIME_DSN" {
		t.Fatalf("expected safe logical references to remain visible, got %+v", redacted)
	}
	if redacted["namespace"] != "prod" {
		t.Fatalf("expected non-sensitive metadata to remain visible, got %+v", redacted)
	}
	headers, ok := redacted["headers"].(Metadata)
	if !ok {
		t.Fatalf("expected nested headers metadata, got %T", redacted["headers"])
	}
	if headers["Authorization"] != RedactedMetadataValue || headers["Accept"] != "application/json" {
		t.Fatalf("expected nested authorization redaction only, got %+v", headers)
	}
	if metadata["access_token"] != "ghp_legacy_plaintext_token" || metadata["runtime_dsn"] != rawDSN {
		t.Fatalf("redaction mutated source metadata: %+v", metadata)
	}
}

func TestFirstUnsafeMetadataPathAllowsReferenceNamesButRejectsRawValues(t *testing.T) {
	t.Parallel()

	if path := FirstUnsafeMetadataPath(Metadata{
		"access_token_env": "CCP_GITHUB_TOKEN",
		"secret_ref":       "prod/github/app/token",
		"dsn_env":          "CCP_DATABASE_DSN",
	}, "metadata"); path != "" {
		t.Fatalf("expected safe reference metadata to pass validation, got unsafe path %q", path)
	}

	if path := FirstUnsafeMetadataPath(Metadata{
		"secret_ref_env": "postgres" + "://db.internal:5432/app?sslmode=disable",
	}, "metadata"); path != "metadata.secret_ref_env" {
		t.Fatalf("expected raw DSN-shaped safe reference value to be rejected, got %q", path)
	}
}

func TestErrorResponseJSONShape(t *testing.T) {
	t.Parallel()

	payload, err := json.Marshal(ErrorResponse{Error: ErrorDetail{Code: "validation_error", Message: "name is required"}})
	if err != nil {
		t.Fatalf("marshal error response: %v", err)
	}
	var decoded struct {
		Error ErrorDetail `json:"error"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if decoded.Error.Code != "validation_error" || decoded.Error.Message != "name is required" {
		t.Fatalf("unexpected error envelope: %+v", decoded.Error)
	}
}
