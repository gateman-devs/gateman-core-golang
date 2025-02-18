package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var SubscriptionPlanOnce = sync.Once{}

var SubscriptionPlanRepository mongo.MongoRepository[entities.SubscriptionPlan]

func SubscriptionPlanRepo() *mongo.MongoRepository[entities.SubscriptionPlan] {
	SubscriptionPlanOnce.Do(func() {
		SubscriptionPlanRepository = mongo.MongoRepository[entities.SubscriptionPlan]{Model: datastore.SubscriptionPlanModel}
	})
	return &SubscriptionPlanRepository
}
