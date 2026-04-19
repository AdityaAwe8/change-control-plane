package types

type CreatePolicyRequest struct {
	OrganizationID string          `json:"organization_id"`
	ProjectID      string          `json:"project_id,omitempty"`
	ServiceID      string          `json:"service_id,omitempty"`
	EnvironmentID  string          `json:"environment_id,omitempty"`
	Name           string          `json:"name"`
	Code           string          `json:"code,omitempty"`
	AppliesTo      string          `json:"applies_to"`
	Mode           string          `json:"mode"`
	Enabled        *bool           `json:"enabled,omitempty"`
	Priority       int             `json:"priority,omitempty"`
	Description    string          `json:"description,omitempty"`
	Conditions     PolicyCondition `json:"conditions,omitempty"`
	Metadata       Metadata        `json:"metadata,omitempty"`
}

type UpdatePolicyRequest struct {
	ProjectID     *string          `json:"project_id,omitempty"`
	ServiceID     *string          `json:"service_id,omitempty"`
	EnvironmentID *string          `json:"environment_id,omitempty"`
	Name          *string          `json:"name,omitempty"`
	Code          *string          `json:"code,omitempty"`
	AppliesTo     *string          `json:"applies_to,omitempty"`
	Mode          *string          `json:"mode,omitempty"`
	Enabled       *bool            `json:"enabled,omitempty"`
	Priority      *int             `json:"priority,omitempty"`
	Description   *string          `json:"description,omitempty"`
	Conditions    *PolicyCondition `json:"conditions,omitempty"`
	Metadata      Metadata         `json:"metadata,omitempty"`
}
