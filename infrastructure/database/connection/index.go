package connection

import (
	"gateman.io/infrastructure/database/connection/cache"
	"gateman.io/infrastructure/database/connection/datastore"
)

func ConnectToDatabase() {
	datastore.ConnectToDatabase()
	cache.ConnectToCache()
}
