package entities

import (
	"time"

	"gateman.io/application/utils"
)

type SubscriptionFrequency string

var Monthly SubscriptionFrequency = "monthly"
var Annually SubscriptionFrequency = "annually"

type SubscriptionPlanName string

var Free SubscriptionPlanName = "Free"
var Essential SubscriptionPlanName = "Essential"
var Premium SubscriptionPlanName = "Premium"

type SubscriptionPlan struct {
	Features     []string             `bson:"features" json:"features"`
	MonthlyPrice uint32               `bson:"monthlyPrice" json:"monthlyPrice"`
	AnnualPrice  uint32               `bson:"annualPrice" json:"annualPrice"`
	Name         SubscriptionPlanName `bson:"name" json:"name"`

	ID        string     `bson:"_id" json:"id"`
	CreatedAt time.Time  `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time  `bson:"updatedAt" json:"updatedAt"`
	DeletedAt *time.Time `bson:"deletedAt" json:"deletedAt"`
}

func (model SubscriptionPlan) ParseModel() any {
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
