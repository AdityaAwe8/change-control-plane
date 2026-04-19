package storage

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *PostgresStore) CreateBrowserSession(ctx context.Context, session types.BrowserSession) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO browser_sessions (
			id, user_id, session_hash, auth_method, auth_provider_id, auth_provider,
			last_seen_at, expires_at, revoked_at, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, session.ID, session.UserID, session.SessionHash, session.AuthMethod, session.AuthProviderID, session.AuthProvider, session.LastSeenAt, session.ExpiresAt, session.RevokedAt, jsonValue(session.Metadata), session.CreatedAt, session.UpdatedAt)
	return err
}

func (s *PostgresStore) GetBrowserSessionByHash(ctx context.Context, sessionHash string) (types.BrowserSession, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, user_id, session_hash, auth_method, auth_provider_id, auth_provider,
			last_seen_at, expires_at, revoked_at, metadata, created_at, updated_at
		FROM browser_sessions
		WHERE session_hash = $1
	`, sessionHash)
	return scanBrowserSession(row)
}

func (s *PostgresStore) UpdateBrowserSession(ctx context.Context, session types.BrowserSession) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE browser_sessions
		SET auth_method = $2,
			auth_provider_id = $3,
			auth_provider = $4,
			last_seen_at = $5,
			expires_at = $6,
			revoked_at = $7,
			metadata = $8,
			updated_at = $9
		WHERE id = $1
	`, session.ID, session.AuthMethod, session.AuthProviderID, session.AuthProvider, session.LastSeenAt, session.ExpiresAt, session.RevokedAt, jsonValue(session.Metadata), session.UpdatedAt)
	return err
}

func scanBrowserSession(scanner interface {
	Scan(dest ...any) error
}) (types.BrowserSession, error) {
	var session types.BrowserSession
	var metadata []byte
	var lastSeenAt sql.NullTime
	var revokedAt sql.NullTime
	if err := scanner.Scan(
		&session.ID,
		&session.UserID,
		&session.SessionHash,
		&session.AuthMethod,
		&session.AuthProviderID,
		&session.AuthProvider,
		&lastSeenAt,
		&session.ExpiresAt,
		&revokedAt,
		&metadata,
		&session.CreatedAt,
		&session.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return types.BrowserSession{}, ErrNotFound
		}
		return types.BrowserSession{}, err
	}
	_ = json.Unmarshal(metadata, &session.Metadata)
	if lastSeenAt.Valid {
		value := lastSeenAt.Time.UTC()
		session.LastSeenAt = &value
	}
	if revokedAt.Valid {
		value := revokedAt.Time.UTC()
		session.RevokedAt = &value
	}
	session.CreatedAt = session.CreatedAt.UTC()
	session.UpdatedAt = session.UpdatedAt.UTC()
	session.ExpiresAt = session.ExpiresAt.UTC()
	return session, nil
}
