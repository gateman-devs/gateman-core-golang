package dto

import "gateman.io/entities"

type CreateHelpRequestDTO struct {
	Summary string `json:"summary" validate:"required,min=10,max=200"`
	Details string `json:"details" validate:"required,min=20,max=2000"`
}

type FetchHelpRequestsDTO struct {
	LastID *string `json:"lastID" validate:"omitempty,ulid"`
	Limit  int64   `json:"limit" validate:"required,min=1,max=100"`
	Status *string `json:"status" validate:"omitempty,oneof=open in-progress resolved closed"`
}

type HelpCenterResponseDTO struct {
	ID        string `json:"id"`
	Summary   string `json:"summary"`
	Details   string `json:"details"`
	Workspace string `json:"workspace"`
	Member    string `json:"member"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

type FetchHelpRequestsResponseDTO struct {
	Data       []entities.HelpCenter `json:"data"`
	HasMore    bool                  `json:"hasMore"`
	NextCursor *string               `json:"nextCursor,omitempty"`
}
