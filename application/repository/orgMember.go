package repository

import (
	"sync"

	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var orgMemberOnce = sync.Once{}

var orgMemberRepository mongo.MongoRepository[entities.OrgMember]

func OrgMemberRepo() *mongo.MongoRepository[entities.OrgMember] {
	orgMemberOnce.Do(func() {
		orgMemberRepository = mongo.MongoRepository[entities.OrgMember]{Model: datastore.OrgMemberModel}
	})
	return &orgMemberRepository
}
