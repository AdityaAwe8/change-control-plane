package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func TestClientSendsAuthOrganizationHeadersAndDecodesListEnvelope(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/integrations" || r.URL.RawQuery != "kind=github&limit=2" {
			t.Fatalf("unexpected request target %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ccpt_test" {
			t.Fatalf("expected bearer token header, got %q", got)
		}
		if got := r.Header.Get("X-CCP-Organization-ID"); got != "org_123" {
			t.Fatalf("expected organization scope header, got %q", got)
		}
		_ = json.NewEncoder(w).Encode(types.ListResponse[types.Integration]{
			Data: []types.Integration{{
				BaseRecord:     types.BaseRecord{ID: "int_123"},
				OrganizationID: "org_123",
				Name:           "GitHub",
				Kind:           "github",
			}},
		})
	}))
	defer server.Close()

	c := New(server.URL + "/")
	c.SetToken(" ccpt_test ")
	c.SetOrganizationID(" org_123 ")

	items, err := c.ListIntegrationsWithQuery(context.Background(), "?kind=github&limit=2")
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}
	if len(items) != 1 || items[0].ID != "int_123" || items[0].Kind != "github" {
		t.Fatalf("unexpected decoded integrations: %+v", items)
	}
}

func TestClientEncodesJSONBodyAndDecodesItemEnvelope(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/integrations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
			t.Fatalf("expected json content type, got %q", got)
		}
		var req types.CreateIntegrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.OrganizationID != "org_123" || req.Kind != "prometheus" || req.Metadata["api_base_url"] != "https://prom.example" {
			t.Fatalf("unexpected create integration payload: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(types.ItemResponse[types.Integration]{
			Data: types.Integration{
				BaseRecord:     types.BaseRecord{ID: "int_prom"},
				OrganizationID: req.OrganizationID,
				Name:           req.Name,
				Kind:           req.Kind,
			},
		})
	}))
	defer server.Close()

	c := New(server.URL)
	created, err := c.CreateIntegration(context.Background(), types.CreateIntegrationRequest{
		OrganizationID: "org_123",
		Kind:           "prometheus",
		Name:           "Prometheus",
		Metadata:       types.Metadata{"api_base_url": "https://prom.example"},
	})
	if err != nil {
		t.Fatalf("create integration: %v", err)
	}
	if created.ID != "int_prom" || created.Kind != "prometheus" {
		t.Fatalf("unexpected created integration: %+v", created)
	}
}

func TestClientListErrorsDecodeRuntimeErrorEnvelope(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(types.ErrorResponse{
			Error: types.ErrorDetail{Code: "forbidden", Message: "active organization is required"},
		})
	}))
	defer server.Close()

	c := New(server.URL)
	_, err := c.ListRepositories(context.Background(), "")
	if err == nil || err.Error() != "active organization is required" {
		t.Fatalf("expected runtime error message, got %v", err)
	}
}
