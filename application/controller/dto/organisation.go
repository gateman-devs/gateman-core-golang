package dto

type CreateOrgDTO struct {
	OrgName string `json:"orgName" validate:"required"`
	Country string `json:"country" validate:"required,iso3166_1_alpha2"`
	Sector  string `json:"sector" validate:"required,oneof=fintech government, health, education other"`
}
