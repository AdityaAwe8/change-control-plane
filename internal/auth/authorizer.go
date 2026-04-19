package auth

import (
	"slices"

	"github.com/change-control-plane/change-control-plane/pkg/types"
)

type Authorizer struct{}

func NewAuthorizer() *Authorizer {
	return &Authorizer{}
}

func (a *Authorizer) CanCreateOrganization(identity Identity) bool {
	return identity.Authenticated && identity.ActorType == types.ActorTypeUser
}

func (a *Authorizer) CanViewOrganization(identity Identity, organizationID string) bool {
	return identity.HasOrganizationAccess(organizationID)
}

func (a *Authorizer) CanManageOrganization(identity Identity, organizationID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin")
}

func (a *Authorizer) CanCreateProject(identity Identity, organizationID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin")
}

func (a *Authorizer) CanManageProject(identity Identity, organizationID, projectID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin")
}

func (a *Authorizer) CanReadProject(identity Identity, organizationID, projectID string) bool {
	return identity.HasOrganizationAccess(organizationID) || identity.HasProjectAccess(projectID)
}

func (a *Authorizer) CanManageService(identity Identity, service types.Service, team types.Team) bool {
	return hasAnyRole(identity.OrganizationRole(service.OrganizationID), "org_admin") ||
		hasAnyRole(identity.ProjectRole(service.ProjectID), "project_admin") ||
		slices.Contains(team.OwnerUserIDs, identity.ActorID)
}

func (a *Authorizer) CanCreateEnvironment(identity Identity, organizationID, projectID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin")
}

func (a *Authorizer) CanCreateTeam(identity Identity, organizationID, projectID string) bool {
	return a.CanCreateEnvironment(identity, organizationID, projectID)
}

func (a *Authorizer) CanManageTeam(identity Identity, organizationID, projectID string) bool {
	return a.CanManageProject(identity, organizationID, projectID)
}

func (a *Authorizer) CanIngestChange(identity Identity, organizationID, projectID string, team types.Team) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin", "org_member") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin", "project_member", "service_owner") ||
		slices.Contains(team.OwnerUserIDs, identity.ActorID)
}

func (a *Authorizer) CanAssessRisk(identity Identity, organizationID, projectID string) bool {
	return a.CanReadProject(identity, organizationID, projectID)
}

func (a *Authorizer) CanPlanRollout(identity Identity, organizationID, projectID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin", "org_member") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin", "project_member", "service_owner")
}

func (a *Authorizer) CanViewAudit(identity Identity, organizationID string) bool {
	return identity.HasOrganizationAccess(organizationID)
}

func (a *Authorizer) CanManageIntegrations(identity Identity, organizationID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin")
}

func (a *Authorizer) CanManageServiceAccounts(identity Identity, organizationID string) bool {
	return identity.ActorType == types.ActorTypeUser && hasAnyRole(identity.OrganizationRole(organizationID), "org_admin")
}

func (a *Authorizer) CanReadServiceAccounts(identity Identity, organizationID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin", "org_member", "viewer")
}

func (a *Authorizer) CanExecuteRollout(identity Identity, organizationID, projectID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin", "org_member") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin", "project_member", "service_owner")
}

func (a *Authorizer) CanRecordVerification(identity Identity, organizationID, projectID string) bool {
	return a.CanExecuteRollout(identity, organizationID, projectID)
}

func (a *Authorizer) CanOverrideRollout(identity Identity, organizationID, projectID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin", "service_owner")
}

func (a *Authorizer) CanManageRollbackPolicies(identity Identity, organizationID, projectID string) bool {
	return a.CanOverrideRollout(identity, organizationID, projectID)
}

func (a *Authorizer) CanManagePolicies(identity Identity, organizationID, projectID string) bool {
	return hasAnyRole(identity.OrganizationRole(organizationID), "org_admin") ||
		hasAnyRole(identity.ProjectRole(projectID), "project_admin")
}

func (a *Authorizer) CanViewPolicies(identity Identity, organizationID, projectID string) bool {
	if projectID == "" {
		return identity.HasOrganizationAccess(organizationID)
	}
	return a.CanReadProject(identity, organizationID, projectID)
}

func (a *Authorizer) CanViewStatusHistory(identity Identity, organizationID, projectID string) bool {
	if projectID == "" {
		return identity.HasOrganizationAccess(organizationID)
	}
	return a.CanReadProject(identity, organizationID, projectID)
}

func hasAnyRole(role string, allowed ...string) bool {
	for _, candidate := range allowed {
		if role == candidate {
			return true
		}
	}
	return false
}
