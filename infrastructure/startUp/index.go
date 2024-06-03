package startup

import (
	polymercore "authone.usepolymer.co/application/services/polymer-core"
	"authone.usepolymer.co/infrastructure/database"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	"authone.usepolymer.co/infrastructure/ipresolver"
	"authone.usepolymer.co/infrastructure/logger"
)

// Used to start services such as loggers, databases, queues, etc.
func StartServices() {
	logger.InitializeLogger()
	database.SetUpDatabase()
	ipresolver.IPResolverInstance.ConnectToDB()
	polymercore.PolymerService.Initialise()
	logger.RequestMetricMonitor.Init()
}

// Used to clean up after services that have been shutdown.
func CleanUpServices() {
	datastore.CleanUp()
}
