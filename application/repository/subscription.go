package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var subscriptionOnce = sync.Once{}

var subscriptionRepository mongo.MongoRepository[entities.ActiveSubscription]

func ActiveSubscriptionRepo() *mongo.MongoRepository[entities.ActiveSubscription] {
	subscriptionOnce.Do(func() {
		subscriptionRepository = mongo.MongoRepository[entities.ActiveSubscription]{Model: datastore.ActiveSubscriptionModel}
	})
	return &subscriptionRepository
}
