package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

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

func (s *PostgresStore) GetBrowserSession(ctx context.Context, id string) (types.BrowserSession, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, user_id, session_hash, auth_method, auth_provider_id, auth_provider,
			last_seen_at, expires_at, revoked_at, metadata, created_at, updated_at
		FROM browser_sessions
		WHERE id = $1
	`, id)
	return scanBrowserSession(row)
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

func (s *PostgresStore) ListBrowserSessions(ctx context.Context, query BrowserSessionQuery) ([]types.BrowserSession, error) {
	args := make([]any, 0, 6)
	conditions := make([]string, 0, 5)
	conditions = append(conditions, "1=1")

	if query.OrganizationID != "" {
		args = append(args, query.OrganizationID)
		placeholder := len(args)
		conditions = append(conditions, `(u.organization_id = $`+strconv.Itoa(placeholder)+` OR EXISTS (
			SELECT 1 FROM organization_memberships om
			WHERE om.user_id = bs.user_id AND om.organization_id = $`+strconv.Itoa(placeholder)+`
		))`)
	}
	if query.UserID != "" {
		args = append(args, query.UserID)
		conditions = append(conditions, "bs.user_id = $"+strconv.Itoa(len(args)))
	}
	if query.Status != "" {
		args = append(args, query.Status)
		statusPlaceholder := "$" + strconv.Itoa(len(args))
		conditions = append(conditions, `(CASE
			WHEN bs.revoked_at IS NOT NULL THEN 'revoked'
			WHEN bs.expires_at < NOW() THEN 'expired'
			ELSE 'active'
		END) = `+statusPlaceholder)
	}

	sqlQuery := `
		SELECT DISTINCT bs.id, bs.user_id, bs.session_hash, bs.auth_method, bs.auth_provider_id, bs.auth_provider,
			bs.last_seen_at, bs.expires_at, bs.revoked_at, bs.metadata, bs.created_at, bs.updated_at
		FROM browser_sessions bs
		JOIN users u ON u.id = bs.user_id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY bs.created_at DESC, bs.id DESC`
	if query.Limit > 0 {
		args = append(args, query.Limit)
		sqlQuery += " LIMIT $" + strconv.Itoa(len(args))
	}
	if query.Offset > 0 {
		args = append(args, query.Offset)
		sqlQuery += " OFFSET $" + strconv.Itoa(len(args))
	}

	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []types.BrowserSession
	for rows.Next() {
		item, err := scanBrowserSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
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
