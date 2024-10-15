package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var appUserOnce = sync.Once{}

var appUserRepository mongo.MongoRepository[entities.AppUser]

func AppUserRepo() *mongo.MongoRepository[entities.AppUser] {
	appUserOnce.Do(func() {
		appUserRepository = mongo.MongoRepository[entities.AppUser]{Model: datastore.AppUserModel}
	})
	return &appUserRepository
}
