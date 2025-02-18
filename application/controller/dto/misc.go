package dto

import (
	"gateman.io/infrastructure/file_upload/types"
)

type GeneratedSignedURLDTO struct {
	Permission   types.SignedURLPermission `json:"permission"`
	AccountImage bool                      `json:"accountImage"`
	FilePath     string                    `json:"filePath"`
}
