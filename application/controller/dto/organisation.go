package dto

type CreateOrgDTO struct {
	WorkspaceName string `json:"WorkspaceName" validate:"required"`
	Country       string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector        string `json:"sector" validate:"required,oneof=fintech government, health, education other"`
}

type UpdateOrgDTO struct {
	WorkspaceName *string `json:"workspaceName" validate:"required"`
	Country       *string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector        *string `json:"sector" validate:"required,oneof=fintech government, health, education other"`
}

type InviteWorspaceMembersDTO struct {
	Emails []string `json:"emails" validate:"required"`
}

type ResendWorspaceInviteDTO struct {
	Email string `json:"email" validate:"required,email,max=100"`
}

type AcknowledgeWorkspaceInviteDTO struct {
	Email    string `json:"email" validate:"required,email,max=100"`
	ID       string `json:"id" validate:"required"`
	Accepted bool   `json:"accepted" validate:"required"`
}
