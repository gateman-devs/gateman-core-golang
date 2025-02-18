package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var appUserOnce = sync.Once{}

var appUserRepository mongo.MongoRepository[entities.AppUser]

func AppUserRepo() *mongo.MongoRepository[entities.AppUser] {
	appUserOnce.Do(func() {
		appUserRepository = mongo.MongoRepository[entities.AppUser]{Model: datastore.AppUserModel}
	})
	return &appUserRepository
}
