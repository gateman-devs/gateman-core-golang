package dto

type SetFaceDTO struct {
	ImagePath string `json:"imagePath" validate:"required"`
}
