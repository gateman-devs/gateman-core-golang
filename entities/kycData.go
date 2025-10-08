package entities

import (
	"time"

	"gateman.io/application/utils"
)

type KYCIdentityData struct {
	UserID string `bson:"userID" json:"userID"`
	IDHash string `bson:"idHash" json:"idHash"`
	Name   string `bson:"name" json:"name"`
	Data   any    `bson:"data" json:"data"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model KYCIdentityData) ParseModel() any {
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
