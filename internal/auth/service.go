package auth

import (
	"context"
	"crypto/subtle"
	"strings"
	"time"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type IdentityStore interface {
	GetUser(context.Context, string) (types.User, error)
	ListOrganizationMembershipsByUser(context.Context, string) ([]types.OrganizationMembership, error)
	ListProjectMembershipsByUser(context.Context, string) ([]types.ProjectMembership, error)
	GetServiceAccount(context.Context, string) (types.ServiceAccount, error)
	GetAPITokenByPrefix(context.Context, string) (types.APIToken, error)
	UpdateAPIToken(context.Context, types.APIToken) error
}

type Service struct {
	store  IdentityStore
	tokens *TokenService
}

func NewService(store IdentityStore, tokens *TokenService) *Service {
	return &Service{store: store, tokens: tokens}
}

func (s *Service) TokenService() *TokenService {
	return s.tokens
}

func (s *Service) LoadIdentity(ctx context.Context, bearerToken, activeOrganizationID string) (Identity, error) {
	token := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(bearerToken), "Bearer "))
	if token == "" {
		return Identity{}, ErrUnauthorized
	}

	if strings.HasPrefix(token, "ccpt_") {
		return s.loadMachineIdentity(ctx, token, activeOrganizationID)
	}

	claims, err := s.tokens.Verify(token)
	if err != nil {
		return Identity{}, ErrUnauthorized
	}

	user, err := s.store.GetUser(ctx, claims.Subject)
	if err != nil {
		return Identity{}, ErrUnauthorized
	}
	orgMemberships, err := s.store.ListOrganizationMembershipsByUser(ctx, user.ID)
	if err != nil {
		return Identity{}, err
	}
	projectMemberships, err := s.store.ListProjectMembershipsByUser(ctx, user.ID)
	if err != nil {
		return Identity{}, err
	}

	identity := Identity{
		Authenticated:           true,
		ActorID:                 user.ID,
		ActorType:               claims.ActorType,
		User:                    user,
		OrganizationMemberships: make(map[string]types.OrganizationMembership, len(orgMemberships)),
		ProjectMemberships:      make(map[string]types.ProjectMembership, len(projectMemberships)),
		OrganizationRoles:       make(map[string]string, len(orgMemberships)),
		ProjectRoles:            make(map[string]string, len(projectMemberships)),
	}

	for _, membership := range orgMemberships {
		identity.OrganizationMemberships[membership.OrganizationID] = membership
		identity.OrganizationRoles[membership.OrganizationID] = membership.Role
	}
	for _, membership := range projectMemberships {
		identity.ProjectMemberships[membership.ProjectID] = membership
		identity.ProjectRoles[membership.ProjectID] = membership.Role
	}
	resolvedOrganizationID, err := resolveActiveOrganization(identity, activeOrganizationID)
	if err != nil {
		return Identity{}, err
	}
	identity.ActiveOrganizationID = resolvedOrganizationID

	return identity, nil
}

func (s *Service) loadMachineIdentity(ctx context.Context, rawToken, activeOrganizationID string) (Identity, error) {
	parts := strings.Split(rawToken, "_")
	if len(parts) < 3 {
		return Identity{}, ErrUnauthorized
	}
	prefix := strings.Join(parts[:2], "_")
	storedToken, err := s.store.GetAPITokenByPrefix(ctx, prefix)
	if err != nil {
		return Identity{}, ErrUnauthorized
	}
	if subtle.ConstantTimeCompare([]byte(storedToken.TokenHash), []byte(s.tokens.HashOpaqueToken(rawToken))) != 1 {
		return Identity{}, ErrUnauthorized
	}
	if storedToken.Status != "active" || storedToken.ServiceAccountID == "" || storedToken.RevokedAt != nil {
		return Identity{}, ErrUnauthorized
	}
	if storedToken.ExpiresAt != nil && time.Now().UTC().After(*storedToken.ExpiresAt) {
		return Identity{}, ErrUnauthorized
	}

	serviceAccount, err := s.store.GetServiceAccount(ctx, storedToken.ServiceAccountID)
	if err != nil {
		return Identity{}, ErrUnauthorized
	}
	if serviceAccount.Status != "active" {
		return Identity{}, ErrUnauthorized
	}

	now := time.Now().UTC()
	storedToken.LastUsedAt = &now
	if err := s.store.UpdateAPIToken(ctx, storedToken); err != nil {
		return Identity{}, err
	}
	serviceAccount.LastUsedAt = &now

	identity := Identity{
		Authenticated:           true,
		ActorID:                 serviceAccount.ID,
		ActorType:               types.ActorTypeServiceAccount,
		ServiceAccount:          serviceAccount,
		OrganizationRoles:       map[string]string{serviceAccount.OrganizationID: serviceAccount.Role},
		ProjectRoles:            map[string]string{},
		OrganizationMemberships: map[string]types.OrganizationMembership{},
		ProjectMemberships:      map[string]types.ProjectMembership{},
	}

	resolvedOrganizationID, err := resolveActiveOrganization(identity, activeOrganizationID)
	if err != nil {
		return Identity{}, err
	}
	identity.ActiveOrganizationID = resolvedOrganizationID
	return identity, nil
}

func resolveActiveOrganization(identity Identity, explicit string) (string, error) {
	if explicit != "" {
		if identity.HasOrganizationAccess(explicit) {
			return explicit, nil
		}
		return "", ErrForbidden
	}
	if identity.ActorType == types.ActorTypeServiceAccount && identity.ServiceAccount.OrganizationID != "" {
		return identity.ServiceAccount.OrganizationID, nil
	}
	if identity.User.OrganizationID != "" && identity.HasOrganizationAccess(identity.User.OrganizationID) {
		return identity.User.OrganizationID, nil
	}
	if ids := identity.OrganizationIDs(); len(ids) == 1 {
		return ids[0], nil
	}
	if ids := identity.OrganizationIDs(); len(ids) > 0 {
		return ids[0], nil
	}
	return "", nil
}
