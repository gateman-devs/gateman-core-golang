package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type SubscriptionPlan struct {
	Features     []string `bson:"features" json:"plfeaturesan"`
	MonthlyPrice string   `bson:"monthlyPrice" json:"monthlyPrice"`
	AnnualPrice  string   `bson:"annualPrice" json:"annualPrice"`
	AnnualURL    string   `bson:"annualURL" json:"annualURL"`
	MonthlyURL   string   `bson:"monthlyURL" json:"monthlyURL"`
	Name         string   `bson:"name" json:"name"`

	ID            string     `bson:"_id" json:"id"`
	CreatedAt     time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt     *time.Time `bson:"deletedAt" json:"deletedAt"`
	DeletedReason *string    `bson:"deletedReason" json:"deletedReason"`
}

func (model SubscriptionPlan) ParseModel() any {
	now := time.Now()
	if model.CreatedAt.IsZero() {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
