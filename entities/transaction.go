package entities

import (
	"time"

	"gateman.io/application/utils"
)

type Transaction struct {
	Amount      uint32  `bson:"amount" json:"amount"`
	RefID       string  `bson:"refID" json:"refID"`
	AppID       *string `bson:"appID" json:"appID"`
	WorkspaceID string  `bson:"workspaceID" json:"workspaceID"`
	PlanID      *string `bson:"planID" json:"planID"`
	Description *string `bson:"description" json:"description"`
	Metadata    any     `bson:"metadata" json:"metadata"`

	ID        string     `bson:"_id" json:"id"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt" json:"deletedAt"`
}

func (model Transaction) ParseModel() any {
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
		if model.ID == "" {
			model.ID = utils.GenerateUULDString()
		}
	}
	model.UpdatedAt = now
	return &model
}
