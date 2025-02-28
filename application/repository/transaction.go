package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var transactionOnce = sync.Once{}

var transactionRepository mongo.MongoRepository[entities.Transaction]

func TransactionRepo() *mongo.MongoRepository[entities.Transaction] {
	transactionOnce.Do(func() {
		transactionRepository = mongo.MongoRepository[entities.Transaction]{Model: datastore.TransactionModel}
	})
	return &transactionRepository
}
