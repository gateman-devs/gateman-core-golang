package dto

type CreateOrgDTO struct {
	FirstName string `bson:"firstName" json:"firstName" validate:"required,name_spacial_char"`
	LastName  string `bson:"lastName" json:"lastName" validate:"required,name_spacial_char"`
	OrgName   string `bson:"orgName" json:"orgName" validate:"required,name_spacial_char"`
	Email     string `bson:"email" json:"email" validate:"required,email"`
	Password  string `bson:"password" json:"password" validate:"required,password"`
	Country   string `bson:"country" json:"country" validate:"required,iso3166_1_alpha2"`
	Sector    string `bson:"sector" json:"sector" validate:"required,oneof=fintech government education other"`
}
