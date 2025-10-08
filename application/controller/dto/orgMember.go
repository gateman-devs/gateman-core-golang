package dto

import "gateman.io/entities"

type AddMemberDTO struct {
	Email       string                       `json:"email" validate:"email,min=6,max=100"`
	FirstName   string                       `json:"firstName" validate:"max=100,min=2"`
	LastName    string                       `json:"lastName" validate:"max=100,min=2"`
	Permissions []entities.MemberPermissions `json:"permissions" validate:"required"`
}
