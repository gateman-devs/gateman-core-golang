package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var userOnce = sync.Once{}

var userRepository mongo.MongoRepository[entities.User]

func UserRepo() *mongo.MongoRepository[entities.User] {
	userOnce.Do(func() {
		userRepository = mongo.MongoRepository[entities.User]{Model: datastore.UserModel}
	})
	return &userRepository
}
