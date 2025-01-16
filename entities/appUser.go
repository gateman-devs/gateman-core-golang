package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

// This represents a user signed up to an app
type AppUser struct {
	AppID         string     `bson:"appID" json:"appID"`
	UserID        string     `bson:"userID" json:"userID"`
	WorkspaceID   string     `bson:"workspaceID" json:"workspaceID"`
	Blocked       bool       `bson:"blocked" json:"blocked"`
	BlockedReason *string    `bson:"blockedReason" json:"blockedReason"`
	BlockedUserAt *time.Time `bson:"blockedUserAt" json:"blockedUserAt"`
	Pin           *string    `bson:"pin" json:"pin"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model AppUser) ParseModel() any {
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
