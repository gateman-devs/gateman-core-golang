package repository

import (
	"sync"

	"gateman.io/entities"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/database/repository/mongo"
)

var workspaceMemberOnce = sync.Once{}

var workspaceMemberRepository mongo.MongoRepository[entities.WorkspaceMember]

func WorkspaceMemberRepo() *mongo.MongoRepository[entities.WorkspaceMember] {
	workspaceMemberOnce.Do(func() {
		workspaceMemberRepository = mongo.MongoRepository[entities.WorkspaceMember]{Model: datastore.WorkspaceMemberModel}
	})
	return &workspaceMemberRepository
}
