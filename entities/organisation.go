package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type Workspace struct {
	Name        string `bson:"name" json:"name"`
	Email       string `bson:"email" json:"email"`
	SuperMember string `bson:"superMember" json:"superMember"`
	CreatedBy   string `bson:"createdBy" json:"createdBy"`
	Country     string `bson:"country" json:"country"`
	Sector      string `bson:"sector" json:"sector"`
	Verified    bool   `bson:"verified" json:"verified"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model Workspace) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
