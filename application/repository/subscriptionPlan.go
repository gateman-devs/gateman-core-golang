package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var subscriptionPlanOnce = sync.Once{}

var SubscriptionPlanRepository mongo.MongoRepository[entities.SubscriptionPlan]

func SubscriptionPlanRepo() *mongo.MongoRepository[entities.SubscriptionPlan] {
	subscriptionPlanOnce.Do(func() {
		SubscriptionPlanRepository = mongo.MongoRepository[entities.SubscriptionPlan]{Model: datastore.SubscriptionPlanModel}
	})
	return &SubscriptionPlanRepository
}
