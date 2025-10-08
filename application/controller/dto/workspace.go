package dto

import "gateman.io/entities"

type CreateWorkspaceDTO struct {
	Name     string `json:"name" validate:"required,min=2,max=100"`
	Email    string `json:"email" validate:"required,email,min=6,max=100"`
	Password string `json:"password" validate:"required,password,max=30"`
	Country  string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector   string `json:"sector" validate:"required,oneof=fintech government health education other"`
}

type LoginWorkspaceMemberDTO struct {
	Email    string `json:"email" validate:"required,email,min=6,max=100"`
	Password string `json:"password" validate:"required,max=30"`
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
	ID string `json:"ID" validate:"required,ulid"`
}

type AcknowledgeWorkspaceInviteDTO struct {
	ID       string `json:"id" validate:"required,ulid"`
	Accepted bool   `json:"accepted" validate:"required"`
}

type ResetWorkspaceMemberPasswordDTO struct {
	Email *string `json:"email" validate:"omitempty,email,min=6,max=100"`
	CurrentPassword string `json:"currentPassword" validate:"required,max=30"`
	NewPassword string `json:"newPassword" validate:"required,max=30"`
}