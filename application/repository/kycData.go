package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var kycIdentityOnce = sync.Once{}

var kycIdentityRepository mongo.MongoRepository[entities.KYCIdentityData]

func KycIdentityRepo() *mongo.MongoRepository[entities.KYCIdentityData] {
	kycIdentityOnce.Do(func() {
		kycIdentityRepository = mongo.MongoRepository[entities.KYCIdentityData]{Model: datastore.KYCIdentityDataModel}
	})
	return &kycIdentityRepository
}
