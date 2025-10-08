package entities

import (
	"time"

	"gateman.io/application/utils"
)

type HelpCenter struct {
	Summary   string `bson:"summary" json:"summary" validate:"required,min=10,max=200"`
	Details   string `bson:"details" json:"details" validate:"required,min=20,max=2000"`
	Workspace string `bson:"workspace" json:"workspace" validate:"required"`
	Member    string `bson:"member" json:"member" validate:"required"`
	Status    string `bson:"status" json:"status"` // open, in-progress, resolved, closed

	ID        string     `bson:"_id" json:"id"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt" json:"deletedAt"`
}

func (model HelpCenter) ParseModel() any {
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
		if model.ID == "" {
			model.ID = utils.GenerateUULDString()
		}
	}
	if model.Status == "" {
		model.Status = "open"
	}
	model.UpdatedAt = now
	return &model
}
