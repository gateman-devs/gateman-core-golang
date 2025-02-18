package startup

import (
	"gateman.io/infrastructure/biometric"
	"gateman.io/infrastructure/database"
	"gateman.io/infrastructure/database/connection/datastore"
	fileupload "gateman.io/infrastructure/file_upload"
	identityverification "gateman.io/infrastructure/identity_verification"
	"gateman.io/infrastructure/logger"
	"gateman.io/infrastructure/messaging/sms"
)

// Used to start services such as loggers, databases, queues, etc.
func StartServices() {
	logger.InitializeLogger()
	database.SetUpDatabase()
	logger.RequestMetricMonitor.Init()
	fileupload.InitialiseFileUploader()
	identityverification.InitialiseIdentityVerifier()
	biometric.InitialiseBiometricService()
	sms.InitSMSService()
}

// Used to clean up after services that have been shutdown.
func CleanUpServices() {
	datastore.CleanUp()
}
