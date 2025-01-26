package startup

import (
	"authone.usepolymer.co/infrastructure/biometric"
	"authone.usepolymer.co/infrastructure/database"
	"authone.usepolymer.co/infrastructure/database/connection/datastore"
	fileupload "authone.usepolymer.co/infrastructure/file_upload"
	identityverification "authone.usepolymer.co/infrastructure/identity_verification"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/messaging/sms"
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
