package types

import "time"

type ActorType string

const (
	ActorTypeUser           ActorType = "user"
	ActorTypeServiceAccount ActorType = "service_account"
)

type OrganizationMembership struct {
	BaseRecord
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
	Status         string `json:"status"`
}

type ProjectMembership struct {
	BaseRecord
	UserID         string `json:"user_id"`
	OrganizationID string `json:"organization_id"`
	ProjectID      string `json:"project_id"`
	Role           string `json:"role"`
	Status         string `json:"status"`
}

type ServiceAccount struct {
	BaseRecord
	OrganizationID  string     `json:"organization_id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Role            string     `json:"role"`
	CreatedByUserID string     `json:"created_by_user_id,omitempty"`
	Status          string     `json:"status"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
}

type APIToken struct {
	BaseRecord
	OrganizationID   string     `json:"organization_id"`
	UserID           string     `json:"user_id,omitempty"`
	ServiceAccountID string     `json:"service_account_id,omitempty"`
	Name             string     `json:"name"`
	TokenPrefix      string     `json:"token_prefix"`
	TokenHash        string     `json:"-"`
	Status           string     `json:"status"`
	LastUsedAt       *time.Time `json:"last_used_at,omitempty"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

type BrowserSession struct {
	BaseRecord
	UserID         string     `json:"user_id"`
	SessionHash    string     `json:"-"`
	AuthMethod     string     `json:"auth_method,omitempty"`
	AuthProviderID string     `json:"auth_provider_id,omitempty"`
	AuthProvider   string     `json:"auth_provider,omitempty"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	ExpiresAt      time.Time  `json:"expires_at"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
}
