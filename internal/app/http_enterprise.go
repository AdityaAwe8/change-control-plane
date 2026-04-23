package app

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *HTTPServer) listPublicIdentityProviders(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListPublicIdentityProviders(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.PublicIdentityProvider]{Data: result})
}

func (s *HTTPServer) startIdentityProviderSignIn(w http.ResponseWriter, r *http.Request) {
	var req types.IdentityProviderStartRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.StartIdentityProviderSignIn(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IdentityProviderStartResult]{Data: result})
}

func (s *HTTPServer) completeIdentityProviderSignIn(w http.ResponseWriter, r *http.Request) {
	result, returnTo, err := s.app.CompleteIdentityProviderSignIn(r.Context(), r.URL.Query().Get("state"), r.URL.Query())
	if err != nil {
		if redirectTo := strings.TrimSpace(returnTo); redirectTo != "" {
			http.Redirect(w, r, mergeAuthRedirect(redirectTo, "", err.Error(), false), http.StatusFound)
			return
		}
		writeAppError(w, err)
		return
	}
	if err := s.issueBrowserSessionCookie(r.Context(), w, result.Session); err != nil {
		writeAppError(w, err)
		return
	}
	redirectTo := strings.TrimSpace(returnTo)
	if redirectTo == "" {
		writeJSON(w, http.StatusOK, types.ItemResponse[types.AuthResponse]{Data: result})
		return
	}
	http.Redirect(w, r, mergeAuthRedirect(redirectTo, result.Session.ActiveOrganizationID, "", true), http.StatusFound)
}

func (s *HTTPServer) listIdentityProviders(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListIdentityProviders(r.Context())
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.IdentityProvider]{Data: result})
}

func (s *HTTPServer) createIdentityProvider(w http.ResponseWriter, r *http.Request) {
	var req types.CreateIdentityProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.CreateIdentityProvider(r.Context(), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, types.ItemResponse[types.IdentityProvider]{Data: result})
}

func (s *HTTPServer) updateIdentityProvider(w http.ResponseWriter, r *http.Request) {
	var req types.UpdateIdentityProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	result, err := s.app.UpdateIdentityProvider(r.Context(), r.PathValue("id"), req)
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IdentityProvider]{Data: result})
}

func (s *HTTPServer) testIdentityProvider(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.TestIdentityProvider(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.IdentityProviderTestResult]{Data: result})
}

func (s *HTTPServer) listBrowserSessions(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListBrowserSessions(r.Context(), storage.BrowserSessionQuery{
		UserID: strings.TrimSpace(r.URL.Query().Get("user_id")),
		Status: strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:  parseIntQuery(r, "limit", 50),
		Offset: parseIntQuery(r, "offset", 0),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.BrowserSessionInfo]{Data: result})
}

func (s *HTTPServer) revokeBrowserSessionByID(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.RevokeBrowserSessionByID(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	if result.Current {
		s.clearBrowserSessionCookie(w)
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.BrowserSessionInfo]{Data: result})
}

func (s *HTTPServer) getWebhookRegistration(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.GetWebhookRegistration(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.WebhookRegistrationResult]{Data: result})
}

func (s *HTTPServer) syncWebhookRegistration(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.EnsureWebhookRegistration(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.WebhookRegistrationResult]{Data: result})
}

func (s *HTTPServer) listOutboxEvents(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.ListOutboxEvents(r.Context(), storage.OutboxEventQuery{
		EventType: strings.TrimSpace(r.URL.Query().Get("event_type")),
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:     parseIntQuery(r, "limit", 100),
		Offset:    parseIntQuery(r, "offset", 0),
	})
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ListResponse[types.OutboxEvent]{Data: result})
}

func (s *HTTPServer) retryOutboxEvent(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.RetryOutboxEvent(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.OutboxEvent]{Data: result})
}

func (s *HTTPServer) requeueOutboxEvent(w http.ResponseWriter, r *http.Request) {
	result, err := s.app.RequeueOutboxEvent(r.Context(), r.PathValue("id"))
	if err != nil {
		writeAppError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, types.ItemResponse[types.OutboxEvent]{Data: result})
}

func mergeAuthRedirect(returnTo, organizationID, message string, completed bool) string {
	parsed, err := url.Parse(returnTo)
	if err != nil {
		return returnTo
	}
	if strings.TrimSpace(parsed.Fragment) != "" {
		fragmentPath := parsed.Fragment
		fragmentQuery := ""
		if idx := strings.Index(fragmentPath, "?"); idx >= 0 {
			fragmentQuery = fragmentPath[idx+1:]
			fragmentPath = fragmentPath[:idx]
		}
		values, parseErr := url.ParseQuery(fragmentQuery)
		if parseErr == nil {
			if strings.TrimSpace(organizationID) != "" {
				values.Set("organization_id", organizationID)
			}
			if strings.TrimSpace(message) != "" {
				values.Set("auth_error", message)
			}
			if completed {
				values.Set("auth_complete", "1")
			}
			parsed.Fragment = fragmentPath
			if encoded := values.Encode(); encoded != "" {
				parsed.Fragment += "?" + encoded
			}
			return parsed.String()
		}
	}
	values := parsed.Query()
	if strings.TrimSpace(organizationID) != "" {
		values.Set("organization_id", organizationID)
	}
	if strings.TrimSpace(message) != "" {
		values.Set("auth_error", message)
	}
	if completed {
		values.Set("auth_complete", "1")
	}
	parsed.RawQuery = values.Encode()
	return parsed.String()
}
