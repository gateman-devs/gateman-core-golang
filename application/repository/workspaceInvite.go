package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var workspaceInviteOnce = sync.Once{}

var workspaceInviteRepository mongo.MongoRepository[entities.WorkspaceInvite]

func WorkspaceInviteRepo() *mongo.MongoRepository[entities.WorkspaceInvite] {
	workspaceInviteOnce.Do(func() {
		workspaceInviteRepository = mongo.MongoRepository[entities.WorkspaceInvite]{Model: datastore.WorkspaceInviteModel}
	})
	return &workspaceInviteRepository
}
