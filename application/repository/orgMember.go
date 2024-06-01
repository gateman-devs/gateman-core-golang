package repository

import (
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var OrgMemberRepository mongo.MongoRepository[entities.OrgMember] = mongo.MongoRepository[entities.OrgMember]{Model: datastore.OrgMemberModel}
