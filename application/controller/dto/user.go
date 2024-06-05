package dto

type CreateUserDTO struct {
	Email    string `bson:"email" json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}
