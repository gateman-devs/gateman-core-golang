package entities

import (
	"time"

	"gateman.io/application/utils"
)

type Workspace struct {
	Name               string     `bson:"name" json:"name"`
	Email              string     `bson:"email" json:"email"`
	VerifiedEmail      bool       `bson:"verifiedEmail" json:"verifiedEmail"`
	SuperMember        string     `bson:"superMember" json:"superMember"`
	Country            string     `bson:"country" json:"country"`
	Sector             string     `bson:"sector" json:"sector"`
	DefaultPaymentCard string     `bson:"defaultPaymentCard" json:"defaultPaymentCard"`
	PaymentDetails     []CardInfo `bson:"paymentDetails" json:"paymentDetails"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model Workspace) ParseModel() any {
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
