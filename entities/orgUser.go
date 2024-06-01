package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

// This represents a user signed up to an organisation
type OrgUser struct {
	OrganisationID string     `bson:"organisationID" json:"organisationID"`
	UserID         string     `bson:"userID" json:"userID"`
	Blocked        bool       `bson:"blocked" json:"blocked"`
	DeletedAt      *time.Time `bson:"deletedAt" json:"deletedAt"`

	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model OrgUser) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
