package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var workspaceMemberOnce = sync.Once{}

var workspaceMemberRepository mongo.MongoRepository[entities.WorkspaceMember]

func WorkspaceMemberRepo() *mongo.MongoRepository[entities.WorkspaceMember] {
	workspaceMemberOnce.Do(func() {
		workspaceMemberRepository = mongo.MongoRepository[entities.WorkspaceMember]{Model: datastore.WorkspaceMemberModel}
	})
	return &workspaceMemberRepository
}
