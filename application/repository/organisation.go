package repository

import (
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/database/repository/mongo"
)

var OrgRepository mongo.MongoRepository[entities.Organisation] = mongo.MongoRepository[entities.Organisation]{Model: datastore.OrgModel}
