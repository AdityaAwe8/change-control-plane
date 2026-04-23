package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/internal/common"
	"github.com/change-control-plane/change-control-plane/internal/storage"
	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (a *Application) IssueBrowserSession(ctx context.Context, session types.SessionInfo) (string, error) {
	if !session.Authenticated || session.ActorID == "" || session.ActorType != string(types.ActorTypeUser) {
		return "", ErrUnauthorized
	}

	rawToken, sessionHash, err := a.Auth.TokenService().GenerateBrowserSessionToken()
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	lastSeenAt := now
	record := types.BrowserSession{
		BaseRecord: types.BaseRecord{
			ID:        common.NewID("sess"),
			CreatedAt: now,
			UpdatedAt: now,
		},
		UserID:         session.ActorID,
		SessionHash:    sessionHash,
		AuthMethod:     session.AuthMethod,
		AuthProviderID: session.AuthProviderID,
		AuthProvider:   session.AuthProvider,
		LastSeenAt:     &lastSeenAt,
		ExpiresAt:      now.Add(a.browserSessionTTL()),
	}
	if err := a.Store.CreateBrowserSession(ctx, record); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (a *Application) RevokeBrowserSession(ctx context.Context, rawToken string) error {
	token := strings.TrimSpace(rawToken)
	if token == "" {
		return nil
	}

	record, err := a.Store.GetBrowserSessionByHash(ctx, a.Auth.TokenService().HashOpaqueToken(token))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil
		}
		return err
	}
	if record.RevokedAt != nil {
		return nil
	}

	now := time.Now().UTC()
	record.RevokedAt = &now
	record.UpdatedAt = now
	if record.LastSeenAt == nil {
		record.LastSeenAt = &now
	}
	return a.Store.UpdateBrowserSession(ctx, record)
}

func (a *Application) ListBrowserSessions(ctx context.Context, query storage.BrowserSessionQuery) ([]types.BrowserSessionInfo, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return nil, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return nil, err
	}
	if !a.Authorizer.CanManageBrowserSessions(identity, orgID) {
		return nil, a.forbidden(ctx, identity, "browser_session.list.denied", "browser_session", "", orgID, "", []string{"actor lacks browser session administration permission"})
	}
	query.OrganizationID = orgID
	if query.Limit <= 0 {
		query.Limit = 50
	}
	sessions, err := a.Store.ListBrowserSessions(ctx, query)
	if err != nil {
		return nil, err
	}
	return a.buildBrowserSessionInfos(ctx, sessions, identity.BrowserSessionID), nil
}

func (a *Application) RevokeBrowserSessionByID(ctx context.Context, id string) (types.BrowserSessionInfo, error) {
	identity, err := a.requireIdentity(ctx)
	if err != nil {
		return types.BrowserSessionInfo{}, err
	}
	orgID, err := a.requireActiveOrganization(identity)
	if err != nil {
		return types.BrowserSessionInfo{}, err
	}
	if !a.Authorizer.CanManageBrowserSessions(identity, orgID) {
		return types.BrowserSessionInfo{}, a.forbidden(ctx, identity, "browser_session.revoke.denied", "browser_session", id, orgID, "", []string{"actor lacks browser session administration permission"})
	}

	record, err := a.Store.GetBrowserSession(ctx, id)
	if err != nil {
		return types.BrowserSessionInfo{}, err
	}
	ok, err := a.browserSessionBelongsToOrganization(ctx, record, orgID)
	if err != nil {
		return types.BrowserSessionInfo{}, err
	}
	if !ok {
		return types.BrowserSessionInfo{}, a.forbidden(ctx, identity, "browser_session.revoke.denied", "browser_session", id, orgID, "", []string{"browser session does not belong to the active organization"})
	}
	if record.RevokedAt == nil {
		now := time.Now().UTC()
		record.RevokedAt = &now
		record.UpdatedAt = now
		if record.LastSeenAt == nil {
			record.LastSeenAt = &now
		}
		if err := a.Store.UpdateBrowserSession(ctx, record); err != nil {
			return types.BrowserSessionInfo{}, err
		}
		if err := a.record(ctx, identity, "browser_session.revoked", "browser_session", record.ID, orgID, "", []string{record.UserID, record.AuthMethod, "revoked"},
			withStatusCategory("auth"),
			withStatusSummary(fmt.Sprintf("browser session revoked for %s", record.UserID)),
		); err != nil {
			return types.BrowserSessionInfo{}, err
		}
	}
	info, err := a.buildBrowserSessionInfo(ctx, record, identity.BrowserSessionID)
	if err != nil {
		return types.BrowserSessionInfo{}, err
	}
	return info, nil
}

func (a *Application) browserSessionTTL() time.Duration {
	minutes := a.Config.BrowserSessionTTL
	if minutes <= 0 {
		minutes = a.Config.AuthTokenTTL
	}
	if minutes <= 0 {
		minutes = 480
	}
	return time.Duration(minutes) * time.Minute
}

func (a *Application) buildBrowserSessionInfos(ctx context.Context, sessions []types.BrowserSession, currentSessionID string) []types.BrowserSessionInfo {
	items := make([]types.BrowserSessionInfo, 0, len(sessions))
	for _, session := range sessions {
		item, err := a.buildBrowserSessionInfo(ctx, session, currentSessionID)
		if err != nil {
			continue
		}
		items = append(items, item)
	}
	return items
}

func (a *Application) buildBrowserSessionInfo(ctx context.Context, session types.BrowserSession, currentSessionID string) (types.BrowserSessionInfo, error) {
	user, err := a.Store.GetUser(ctx, session.UserID)
	if err != nil {
		return types.BrowserSessionInfo{}, err
	}
	now := time.Now().UTC()
	return types.BrowserSessionInfo{
		BaseRecord:      session.BaseRecord,
		UserID:          session.UserID,
		UserEmail:       user.Email,
		UserDisplayName: user.DisplayName,
		AuthMethod:      session.AuthMethod,
		AuthProviderID:  session.AuthProviderID,
		AuthProvider:    session.AuthProvider,
		LastSeenAt:      session.LastSeenAt,
		ExpiresAt:       session.ExpiresAt,
		RevokedAt:       session.RevokedAt,
		Status:          browserSessionStatus(session, now),
		Current:         currentSessionID != "" && session.ID == currentSessionID,
	}, nil
}

func (a *Application) browserSessionBelongsToOrganization(ctx context.Context, session types.BrowserSession, orgID string) (bool, error) {
	user, err := a.Store.GetUser(ctx, session.UserID)
	if err != nil {
		return false, err
	}
	if user.OrganizationID == orgID {
		return true, nil
	}
	memberships, err := a.Store.ListOrganizationMembershipsByUser(ctx, session.UserID)
	if err != nil {
		return false, err
	}
	for _, membership := range memberships {
		if membership.OrganizationID == orgID {
			return true, nil
		}
	}
	return false, nil
}
