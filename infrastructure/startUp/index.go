package startup

import (
	"gateman.io/infrastructure/database"
	"gateman.io/infrastructure/database/connection/datastore"
	fileupload "gateman.io/infrastructure/file_upload"
	identityverification "gateman.io/infrastructure/identity_verification"
	"gateman.io/infrastructure/logger"
	"gateman.io/infrastructure/messaging/sms"
	"gateman.io/infrastructure/payments"
)

// Used to start services such as loggers, databases, queues, etc.
func StartServices() {
	logger.InitializeLogger()
	database.SetUpDatabase()
	logger.RequestMetricMonitor.Init()
	fileupload.InitialiseFileUploader()
	identityverification.InitialiseIdentityVerifier()
	sms.InitSMSService()
	payments.InitialisePaymentProcessor()
}

// Used to clean up after services that have been shutdown.
func CleanUpServices() {
	datastore.CleanUp()
}
