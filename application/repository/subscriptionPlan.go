package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var SubscriptionPlanOnce = sync.Once{}

var SubscriptionPlanRepository mongo.MongoRepository[entities.SubscriptionPlan]

func SubscriptionPlanRepo() *mongo.MongoRepository[entities.SubscriptionPlan] {
	SubscriptionPlanOnce.Do(func() {
		SubscriptionPlanRepository = mongo.MongoRepository[entities.SubscriptionPlan]{Model: datastore.SubscriptionPlanModel}
	})
	return &SubscriptionPlanRepository
}
