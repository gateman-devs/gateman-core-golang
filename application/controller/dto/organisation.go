package dto

import "gateman.io/entities"

type CreateOrgDTO struct {
	WorkspaceName string `json:"workspaceName" validate:"required,min=2,max=100"`
	Country       string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector        string `json:"sector" validate:"required,oneof=fintech government health education other"`
}

type UpdateOrgDTO struct {
	WorkspaceName *string `json:"workspaceName" validate:"required"`
	Country       *string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector        *string `json:"sector" validate:"required,oneof=fintech government health education other"`
}

type MemberInvite struct {
	Email       string                       `json:"email" validate:"required,email,min=6,max=100"`
	Permissions []entities.MemberPermissions `json:"permissions" validate:"required"`
}

type InviteWorspaceMembersDTO struct {
	Invites []MemberInvite `json:"invites" validate:"required"`
}

type ResendWorspaceInviteDTO struct {
	ID string `json:"ID" validate:"required,eq=26"`
}

type AcknowledgeWorkspaceInviteDTO struct {
	ID       string `json:"id" validate:"required,eq=26"`
	Accepted bool   `json:"accepted" validate:"required"`
}
