package dto

type CreateOrgDTO struct {
	FirstName string `json:"firstName" validate:"required,name_spacial_char"`
	LastName  string `json:"lastName" validate:"required,name_spacial_char"`
	OrgName   string `json:"orgName" validate:"required"`
	Email     string `bson:"email" json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=6"`
	Country   string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector    string `json:"sector" validate:"required,oneof=fintech government education other"`
}
