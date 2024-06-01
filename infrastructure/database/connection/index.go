package connection

import (
	"authone.usepolymer.co/infrastructure/database/connection/cache"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
)

func ConnectToDatabase() {
	datastore.ConnectToDatabase()
	cache.ConnectToCache()
}
