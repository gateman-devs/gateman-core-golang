package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var applicationOnce = sync.Once{}

var applicationRepository mongo.MongoRepository[entities.Application]

func ApplicationRepo() *mongo.MongoRepository[entities.Application] {
	applicationOnce.Do(func() {
		applicationRepository = mongo.MongoRepository[entities.Application]{Model: datastore.ApplicationModel}
	})
	return &applicationRepository
}
