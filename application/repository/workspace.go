package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var workspaceOnce = sync.Once{}

var workspaceRepository mongo.MongoRepository[entities.Workspace]

func WorkspaceRepository() *mongo.MongoRepository[entities.Workspace] {
	workspaceOnce.Do(func() {
		workspaceRepository = mongo.MongoRepository[entities.Workspace]{Model: datastore.WorkspaceModel}
	})
	return &workspaceRepository
}
