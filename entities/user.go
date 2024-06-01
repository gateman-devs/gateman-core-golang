package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

// This represents a user signed up to authone
type User struct {
	PolymerID   string   `bson:"polymerID" json:"polymerID"`
	AllowedOrgs []string `bson:"allowedOrgs" json:"allowedOrgs"`

	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model User) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
