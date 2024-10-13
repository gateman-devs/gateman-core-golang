package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var workspaceOnce = sync.Once{}

var workspaceRepository mongo.MongoRepository[entities.Workspace]

func WorkspaceRepository() *mongo.MongoRepository[entities.Workspace] {
	workspaceOnce.Do(func() {
		workspaceRepository = mongo.MongoRepository[entities.Workspace]{Model: datastore.WorkspaceModel}
	})
	return &workspaceRepository
}
