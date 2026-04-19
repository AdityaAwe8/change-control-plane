package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

func (s *PostgresStore) CreateWebhookRegistration(ctx context.Context, registration types.WebhookRegistration) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO webhook_registrations (
			id, organization_id, integration_id, provider_kind, scope_identifier, callback_url, external_hook_id,
			status, delivery_health, auto_managed, last_registered_at, last_validated_at, last_delivery_at,
			last_error, failure_count, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, registration.ID, registration.OrganizationID, registration.IntegrationID, registration.ProviderKind, registration.ScopeIdentifier, registration.CallbackURL, registration.ExternalHookID, registration.Status, registration.DeliveryHealth, registration.AutoManaged, registration.LastRegisteredAt, registration.LastValidatedAt, registration.LastDeliveryAt, registration.LastError, registration.FailureCount, jsonValue(registration.Metadata), registration.CreatedAt, registration.UpdatedAt)
	return err
}

func (s *PostgresStore) GetWebhookRegistrationByIntegration(ctx context.Context, integrationID string) (types.WebhookRegistration, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, integration_id, provider_kind, scope_identifier, callback_url, external_hook_id,
			status, delivery_health, auto_managed, last_registered_at, last_validated_at, last_delivery_at,
			last_error, failure_count, metadata, created_at, updated_at
		FROM webhook_registrations
		WHERE integration_id = $1
	`, integrationID)
	return scanWebhookRegistration(row)
}

func (s *PostgresStore) ListWebhookRegistrations(ctx context.Context, query WebhookRegistrationQuery) ([]types.WebhookRegistration, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, integration_id, provider_kind, scope_identifier, callback_url, external_hook_id,
			status, delivery_health, auto_managed, last_registered_at, last_validated_at, last_delivery_at,
			last_error, failure_count, metadata, created_at, updated_at
		FROM webhook_registrations`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("integration_id", query.IntegrationID),
		filterEqual("provider_kind", query.ProviderKind),
		filterEqual("status", query.Status),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.WebhookRegistration
	for rows.Next() {
		item, err := scanWebhookRegistration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateWebhookRegistration(ctx context.Context, registration types.WebhookRegistration) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE webhook_registrations
		SET scope_identifier = $2, callback_url = $3, external_hook_id = $4, status = $5, delivery_health = $6,
			auto_managed = $7, last_registered_at = $8, last_validated_at = $9, last_delivery_at = $10,
			last_error = $11, failure_count = $12, metadata = $13, updated_at = $14
		WHERE id = $1
	`, registration.ID, registration.ScopeIdentifier, registration.CallbackURL, registration.ExternalHookID, registration.Status, registration.DeliveryHealth, registration.AutoManaged, registration.LastRegisteredAt, registration.LastValidatedAt, registration.LastDeliveryAt, registration.LastError, registration.FailureCount, jsonValue(registration.Metadata), registration.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateIdentityProvider(ctx context.Context, provider types.IdentityProvider) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO identity_providers (
			id, organization_id, name, kind, issuer_url, authorization_endpoint, token_endpoint, userinfo_endpoint,
			jwks_uri, client_id, client_secret_env, scopes, claim_mappings, role_mappings, allowed_domains,
			default_role, enabled, status, connection_health, last_tested_at, last_error, last_authenticated_at,
			metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25)
	`, provider.ID, provider.OrganizationID, provider.Name, provider.Kind, provider.IssuerURL, provider.AuthorizationEndpoint, provider.TokenEndpoint, provider.UserInfoEndpoint, provider.JWKSURI, provider.ClientID, provider.ClientSecretEnv, jsonValue(provider.Scopes), jsonValue(provider.ClaimMappings), jsonValue(provider.RoleMappings), jsonValue(provider.AllowedDomains), provider.DefaultRole, provider.Enabled, provider.Status, provider.ConnectionHealth, provider.LastTestedAt, provider.LastError, provider.LastAuthenticatedAt, jsonValue(provider.Metadata), provider.CreatedAt, provider.UpdatedAt)
	return err
}

func (s *PostgresStore) GetIdentityProvider(ctx context.Context, id string) (types.IdentityProvider, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, name, kind, issuer_url, authorization_endpoint, token_endpoint, userinfo_endpoint,
			jwks_uri, client_id, client_secret_env, scopes, claim_mappings, role_mappings, allowed_domains,
			default_role, enabled, status, connection_health, last_tested_at, last_error, last_authenticated_at,
			metadata, created_at, updated_at
		FROM identity_providers
		WHERE id = $1
	`, id)
	return scanIdentityProvider(row)
}

func (s *PostgresStore) ListIdentityProviders(ctx context.Context, query IdentityProviderQuery) ([]types.IdentityProvider, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, name, kind, issuer_url, authorization_endpoint, token_endpoint, userinfo_endpoint,
			jwks_uri, client_id, client_secret_env, scopes, claim_mappings, role_mappings, allowed_domains,
			default_role, enabled, status, connection_health, last_tested_at, last_error, last_authenticated_at,
			metadata, created_at, updated_at
		FROM identity_providers`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("kind", query.Kind),
		filterOptionalBool("enabled", query.Enabled),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.IdentityProvider
	for rows.Next() {
		item, err := scanIdentityProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateIdentityProvider(ctx context.Context, provider types.IdentityProvider) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE identity_providers
		SET name = $2, issuer_url = $3, authorization_endpoint = $4, token_endpoint = $5, userinfo_endpoint = $6,
			jwks_uri = $7, client_id = $8, client_secret_env = $9, scopes = $10, claim_mappings = $11,
			role_mappings = $12, allowed_domains = $13, default_role = $14, enabled = $15, status = $16,
			connection_health = $17, last_tested_at = $18, last_error = $19, last_authenticated_at = $20,
			metadata = $21, updated_at = $22
		WHERE id = $1
	`, provider.ID, provider.Name, provider.IssuerURL, provider.AuthorizationEndpoint, provider.TokenEndpoint, provider.UserInfoEndpoint, provider.JWKSURI, provider.ClientID, provider.ClientSecretEnv, jsonValue(provider.Scopes), jsonValue(provider.ClaimMappings), jsonValue(provider.RoleMappings), jsonValue(provider.AllowedDomains), provider.DefaultRole, provider.Enabled, provider.Status, provider.ConnectionHealth, provider.LastTestedAt, provider.LastError, provider.LastAuthenticatedAt, jsonValue(provider.Metadata), provider.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateIdentityLink(ctx context.Context, link types.IdentityLink) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO identity_links (
			id, organization_id, provider_id, user_id, external_subject, email, status, last_login_at, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, link.ID, link.OrganizationID, link.ProviderID, link.UserID, link.ExternalSubject, link.Email, link.Status, link.LastLoginAt, jsonValue(link.Metadata), link.CreatedAt, link.UpdatedAt)
	return err
}

func (s *PostgresStore) GetIdentityLinkBySubject(ctx context.Context, providerID, subject string) (types.IdentityLink, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, organization_id, provider_id, user_id, external_subject, email, status, last_login_at, metadata, created_at, updated_at
		FROM identity_links
		WHERE provider_id = $1 AND external_subject = $2
	`, providerID, subject)
	return scanIdentityLink(row)
}

func (s *PostgresStore) ListIdentityLinksByUser(ctx context.Context, userID string) ([]types.IdentityLink, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, organization_id, provider_id, user_id, external_subject, email, status, last_login_at, metadata, created_at, updated_at
		FROM identity_links`,
		0,
		0,
		filterEqual("user_id", userID),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.IdentityLink
	for rows.Next() {
		item, err := scanIdentityLink(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateIdentityLink(ctx context.Context, link types.IdentityLink) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE identity_links
		SET email = $2, status = $3, last_login_at = $4, metadata = $5, updated_at = $6
		WHERE id = $1
	`, link.ID, link.Email, link.Status, link.LastLoginAt, jsonValue(link.Metadata), link.UpdatedAt)
	return err
}

func (s *PostgresStore) CreateOutboxEvent(ctx context.Context, event types.OutboxEvent) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		INSERT INTO outbox_events (
			id, event_type, organization_id, project_id, resource_type, resource_id, status, attempts,
			next_attempt_at, claimed_at, processed_at, last_error, metadata, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, event.ID, event.EventType, nullIfEmpty(event.OrganizationID), nullIfEmpty(event.ProjectID), event.ResourceType, event.ResourceID, event.Status, event.Attempts, event.NextAttemptAt, event.ClaimedAt, event.ProcessedAt, event.LastError, jsonValue(event.Metadata), event.CreatedAt, event.UpdatedAt)
	return err
}

func (s *PostgresStore) GetOutboxEvent(ctx context.Context, id string) (types.OutboxEvent, error) {
	row := s.runner(ctx).QueryRowContext(ctx, `
		SELECT id, event_type, organization_id, project_id, resource_type, resource_id, status, attempts,
			next_attempt_at, claimed_at, processed_at, last_error, metadata, created_at, updated_at
		FROM outbox_events
		WHERE id = $1
	`, id)
	return scanOutboxEvent(row)
}

func (s *PostgresStore) ListOutboxEvents(ctx context.Context, query OutboxEventQuery) ([]types.OutboxEvent, error) {
	sqlQuery, args := buildListQuery(
		`SELECT id, event_type, organization_id, project_id, resource_type, resource_id, status, attempts,
			next_attempt_at, claimed_at, processed_at, last_error, metadata, created_at, updated_at
		FROM outbox_events`,
		query.Limit,
		query.Offset,
		filterEqual("organization_id", query.OrganizationID),
		filterEqual("event_type", query.EventType),
		filterEqual("status", query.Status),
	)
	rows, err := s.runner(ctx).QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.OutboxEvent
	for rows.Next() {
		item, err := scanOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) ClaimOutboxEvents(ctx context.Context, now time.Time, limit int, staleClaimBefore time.Time) ([]types.OutboxEvent, error) {
	rows, err := s.runner(ctx).QueryContext(ctx, `
		UPDATE outbox_events
		SET status = 'processing', claimed_at = $1, updated_at = $1
		WHERE id IN (
			SELECT id
			FROM outbox_events
			WHERE status NOT IN ('processed', 'dead_letter')
				AND (next_attempt_at IS NULL OR next_attempt_at <= $1)
				AND (claimed_at IS NULL OR claimed_at < $2)
			ORDER BY created_at ASC
			LIMIT $3
			FOR UPDATE
		)
		RETURNING id, event_type, organization_id, project_id, resource_type, resource_id, status, attempts,
			next_attempt_at, claimed_at, processed_at, last_error, metadata, created_at, updated_at
	`, now, staleClaimBefore, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []types.OutboxEvent
	for rows.Next() {
		item, err := scanOutboxEvent(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *PostgresStore) UpdateOutboxEvent(ctx context.Context, event types.OutboxEvent) error {
	_, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE outbox_events
		SET status = $2, attempts = $3, next_attempt_at = $4, claimed_at = $5, processed_at = $6,
			last_error = $7, metadata = $8, updated_at = $9
		WHERE id = $1
	`, event.ID, event.Status, event.Attempts, event.NextAttemptAt, event.ClaimedAt, event.ProcessedAt, event.LastError, jsonValue(event.Metadata), event.UpdatedAt)
	return err
}

func (s *PostgresStore) UpdateOutboxEventIfStatus(ctx context.Context, event types.OutboxEvent, expectedStatus string) (bool, error) {
	result, err := s.runner(ctx).ExecContext(ctx, `
		UPDATE outbox_events
		SET status = $2, attempts = $3, next_attempt_at = $4, claimed_at = $5, processed_at = $6,
			last_error = $7, metadata = $8, updated_at = $9
		WHERE id = $1 AND status = $10
	`, event.ID, event.Status, event.Attempts, event.NextAttemptAt, event.ClaimedAt, event.ProcessedAt, event.LastError, jsonValue(event.Metadata), event.UpdatedAt, expectedStatus)
	if err != nil {
		return false, err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return rowsAffected == 1, nil
}

func scanWebhookRegistration(row scanner) (types.WebhookRegistration, error) {
	var item types.WebhookRegistration
	var metadata []byte
	var lastRegisteredAt sql.NullTime
	var lastValidatedAt sql.NullTime
	var lastDeliveryAt sql.NullTime
	err := row.Scan(&item.ID, &item.OrganizationID, &item.IntegrationID, &item.ProviderKind, &item.ScopeIdentifier, &item.CallbackURL, &item.ExternalHookID, &item.Status, &item.DeliveryHealth, &item.AutoManaged, &lastRegisteredAt, &lastValidatedAt, &lastDeliveryAt, &item.LastError, &item.FailureCount, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	if lastRegisteredAt.Valid {
		item.LastRegisteredAt = &lastRegisteredAt.Time
	}
	if lastValidatedAt.Valid {
		item.LastValidatedAt = &lastValidatedAt.Time
	}
	if lastDeliveryAt.Valid {
		item.LastDeliveryAt = &lastDeliveryAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanIdentityProvider(row scanner) (types.IdentityProvider, error) {
	var item types.IdentityProvider
	var scopes []byte
	var claimMappings []byte
	var roleMappings []byte
	var allowedDomains []byte
	var metadata []byte
	var lastTestedAt sql.NullTime
	var lastAuthenticatedAt sql.NullTime
	err := row.Scan(&item.ID, &item.OrganizationID, &item.Name, &item.Kind, &item.IssuerURL, &item.AuthorizationEndpoint, &item.TokenEndpoint, &item.UserInfoEndpoint, &item.JWKSURI, &item.ClientID, &item.ClientSecretEnv, &scopes, &claimMappings, &roleMappings, &allowedDomains, &item.DefaultRole, &item.Enabled, &item.Status, &item.ConnectionHealth, &lastTestedAt, &item.LastError, &lastAuthenticatedAt, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	if lastTestedAt.Valid {
		item.LastTestedAt = &lastTestedAt.Time
	}
	if lastAuthenticatedAt.Valid {
		item.LastAuthenticatedAt = &lastAuthenticatedAt.Time
	}
	_ = json.Unmarshal(scopes, &item.Scopes)
	_ = json.Unmarshal(claimMappings, &item.ClaimMappings)
	_ = json.Unmarshal(roleMappings, &item.RoleMappings)
	_ = json.Unmarshal(allowedDomains, &item.AllowedDomains)
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanIdentityLink(row scanner) (types.IdentityLink, error) {
	var item types.IdentityLink
	var lastLoginAt sql.NullTime
	var metadata []byte
	err := row.Scan(&item.ID, &item.OrganizationID, &item.ProviderID, &item.UserID, &item.ExternalSubject, &item.Email, &item.Status, &lastLoginAt, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	if lastLoginAt.Valid {
		item.LastLoginAt = &lastLoginAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func scanOutboxEvent(row scanner) (types.OutboxEvent, error) {
	var item types.OutboxEvent
	var organizationID sql.NullString
	var projectID sql.NullString
	var nextAttemptAt sql.NullTime
	var claimedAt sql.NullTime
	var processedAt sql.NullTime
	var metadata []byte
	err := row.Scan(&item.ID, &item.EventType, &organizationID, &projectID, &item.ResourceType, &item.ResourceID, &item.Status, &item.Attempts, &nextAttemptAt, &claimedAt, &processedAt, &item.LastError, &metadata, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return item, normalizeNotFound(err)
	}
	item.OrganizationID = organizationID.String
	item.ProjectID = projectID.String
	if nextAttemptAt.Valid {
		item.NextAttemptAt = &nextAttemptAt.Time
	}
	if claimedAt.Valid {
		item.ClaimedAt = &claimedAt.Time
	}
	if processedAt.Valid {
		item.ProcessedAt = &processedAt.Time
	}
	_ = json.Unmarshal(metadata, &item.Metadata)
	return item, nil
}

func filterOptionalBool(column string, value *bool) condition {
	if value == nil {
		return condition{}
	}
	return condition{
		sql:    column + " = $1",
		values: []any{*value},
	}
}
