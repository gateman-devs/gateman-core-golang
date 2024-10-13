package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var workspaceInviteOnce = sync.Once{}

var workspaceInviteRepository mongo.MongoRepository[entities.WorkspaceInvite]

func WorkspaceInviteRepo() *mongo.MongoRepository[entities.WorkspaceInvite] {
	workspaceInviteOnce.Do(func() {
		workspaceInviteRepository = mongo.MongoRepository[entities.WorkspaceInvite]{Model: datastore.WorkspaceInviteModel}
	})
	return &workspaceInviteRepository
}
