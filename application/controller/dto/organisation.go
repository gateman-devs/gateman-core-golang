package dto

import "authone.usepolymer.co/entities"

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

type MemberInvite struct {
	Email       string                       `json:"email" validate:"required,email"`
	Permissions []entities.MemberPermissions `json:"permissions" validate:"required"`
}

type InviteWorspaceMembersDTO struct {
	Invites []MemberInvite `json:"invites" validate:"required"`
}

type ResendWorspaceInviteDTO struct {
	Email string `json:"email" validate:"required,email,max=100"`
}

type AcknowledgeWorkspaceInviteDTO struct {
	Email           string `json:"email" validate:"required,email,max=100"`
	ID              string `json:"id" validate:"required"`
	Accepted        bool   `json:"accepted" validate:"required"`
	ClientPublicKey string `json:"clientPublicKey" validate:"required"`
}
