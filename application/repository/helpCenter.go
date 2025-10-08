package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var helpCenterOnce = sync.Once{}

var helpCenterRepository mongo.MongoRepository[entities.HelpCenter]

func HelpCenterRepository() *mongo.MongoRepository[entities.HelpCenter] {
	helpCenterOnce.Do(func() {
		helpCenterRepository = mongo.MongoRepository[entities.HelpCenter]{Model: datastore.HelpCenterModel}
	})
	return &helpCenterRepository
}
