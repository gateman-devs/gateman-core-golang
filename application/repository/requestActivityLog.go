package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var requestActivityLogOnce = sync.Once{}

var requestActivityLogRepository mongo.MongoRepository[entities.RequestActivityLog]

func RequestActivityLogRepo() *mongo.MongoRepository[entities.RequestActivityLog] {
	requestActivityLogOnce.Do(func() {
		requestActivityLogRepository = mongo.MongoRepository[entities.RequestActivityLog]{Model: datastore.RequestActivityLogModel}
	})
	return &requestActivityLogRepository
}
