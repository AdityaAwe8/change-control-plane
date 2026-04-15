package auth

import (
	"context"
	"errors"
	"sort"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

type Identity struct {
	Authenticated           bool
	ActorID                 string
	ActorType               types.ActorType
	User                    types.User
	ServiceAccount          types.ServiceAccount
	ActiveOrganizationID    string
	OrganizationMemberships map[string]types.OrganizationMembership
	ProjectMemberships      map[string]types.ProjectMembership
	OrganizationRoles       map[string]string
	ProjectRoles            map[string]string
}

func (i Identity) ActorLabel() string {
	if i.ActorType == types.ActorTypeServiceAccount {
		if i.ServiceAccount.Name != "" {
			return i.ServiceAccount.Name
		}
	}
	if i.User.DisplayName != "" {
		return i.User.DisplayName
	}
	if i.User.Email != "" {
		return i.User.Email
	}
	return "system"
}

func (i Identity) OrganizationRole(orgID string) string {
	if role, ok := i.OrganizationRoles[orgID]; ok {
		return role
	}
	if membership, ok := i.OrganizationMemberships[orgID]; ok {
		return membership.Role
	}
	return ""
}

func (i Identity) ProjectRole(projectID string) string {
	if role, ok := i.ProjectRoles[projectID]; ok {
		return role
	}
	if membership, ok := i.ProjectMemberships[projectID]; ok {
		return membership.Role
	}
	return ""
}

func (i Identity) HasOrganizationAccess(orgID string) bool {
	if _, ok := i.OrganizationRoles[orgID]; ok {
		return true
	}
	_, ok := i.OrganizationMemberships[orgID]
	return ok
}

func (i Identity) HasProjectAccess(projectID string) bool {
	if _, ok := i.ProjectRoles[projectID]; ok {
		return true
	}
	_, ok := i.ProjectMemberships[projectID]
	return ok
}

func (i Identity) OrganizationIDs() []string {
	ids := make([]string, 0, len(i.OrganizationMemberships)+len(i.OrganizationRoles))
	seen := make(map[string]struct{}, len(i.OrganizationMemberships)+len(i.OrganizationRoles))
	for id := range i.OrganizationMemberships {
		ids = append(ids, id)
		seen[id] = struct{}{}
	}
	for id := range i.OrganizationRoles {
		if _, ok := seen[id]; ok {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

type identityKey struct{}

func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey{}).(Identity)
	return identity, ok
}
