package entities

import (
	"time"

	"authone.usepolymer.co/application/utils"
)

type SubscriptionPlan struct {
	Features     []string `bson:"features" json:"plfeaturesan"`
	MonthlyPrice string   `bson:"monthlyPrice" json:"monthlyPrice"`
	AnnualPrice  string   `bson:"annualPrice" json:"annualPrice"`
	Name         string   `bson:"name" json:"name"`

	ID        string    `bson:"_id" json:"id"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

func (model SubscriptionPlan) ParseModel() any {
	now := time.Now()
	if model.ID == "" {
		model.CreatedAt = now
		model.ID = utils.GenerateUULDString()
	}
	model.UpdatedAt = now
	return &model
}
