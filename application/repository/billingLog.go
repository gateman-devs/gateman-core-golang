package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var billingLogOnce = sync.Once{}

var billingLogRepository mongo.MongoRepository[entities.BillingLog]

func BillingLogRepository() *mongo.MongoRepository[entities.BillingLog] {
	billingLogOnce.Do(func() {
		billingLogRepository = mongo.MongoRepository[entities.BillingLog]{Model: datastore.BillingLogModel}
	})
	return &billingLogRepository
}
