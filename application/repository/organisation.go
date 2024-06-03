package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var orgOnce = sync.Once{}

var orgRepository mongo.MongoRepository[entities.Organisation]

func OrgRepo() *mongo.MongoRepository[entities.Organisation] {
	orgOnce.Do(func() {
		orgRepository = mongo.MongoRepository[entities.Organisation]{Model: datastore.OrgModel}
	})
	return &orgRepository
}
