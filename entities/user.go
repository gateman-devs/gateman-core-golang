package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

// This represents a user signed up to authone
type User struct {
	PolymerID     string   `bson:"polymerID" json:"polymerID"`
	AllowedOrgs   []string `bson:"allowedOrgs" json:"allowedOrgs"`
	Email         string   `bson:"email" json:"email"`
	Password      string   `bson:"password" json:"password"`
	UserAgent     string   `bson:"userAgent" json:"userAgent"`
	DeviceID      string   `bson:"deviceID" json:"deviceID"`
	Deactivated   bool     `bson:"deactivated" json:"deactivated"`
	VerifiedEmail bool     `bson:"verifiedEmail" json:"verifiedEmail"`

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
