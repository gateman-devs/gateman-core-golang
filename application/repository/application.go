package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var applicationOnce = sync.Once{}

var applicationRepository mongo.MongoRepository[entities.Application]

func ApplicationRepo() *mongo.MongoRepository[entities.Application] {
	applicationOnce.Do(func() {
		applicationRepository = mongo.MongoRepository[entities.Application]{Model: datastore.ApplicationModel}
	})
	return &applicationRepository
}
