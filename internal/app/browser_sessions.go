package app

import (
	"context"
	"errors"
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
