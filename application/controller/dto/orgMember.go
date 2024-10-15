package dto

import "authone.usepolymer.co/entities"

type AddMemberDTO struct {
	Email       string                       `json:"email" validate:"email,max=100"`
	FirstName   string                       `json:"firstName" validate:"max=100,min=2"`
	LastName    string                       `json:"lastName" validate:"max=100,min=2"`
	Permissions []entities.MemberPermissions `json:"permissions" validate:"required"`
}
