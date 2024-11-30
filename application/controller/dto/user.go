package dto

type SetFaceDTO struct {
	ImagePath string `json:"imagePath" validate:"required"`
}

type SetNINDetails struct {
	NIN string `json:"nin" validate:"required"`
}