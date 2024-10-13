package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type ActiveSubscription struct {
	Plan          string `bson:"plan" json:"plan"`
	Active        bool   `bson:"active" json:"active"`
	ApplicationID string `bson:"applicationID" json:"applicationID"`
	Name          string `bson:"name" json:"name"`
	RenewalDate   string `bson:"renewalDate" json:"renewalDate"`
	Payment       string `bson:"payment" json:"payment"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model ActiveSubscription) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
