package startup

import (
	"gateman.io/infrastructure/biometric"
	"gateman.io/infrastructure/database"
	"gateman.io/infrastructure/database/connection/datastore"
	"gateman.io/infrastructure/facematch"
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
	biometric.InitialiseBiometricService()

	// Initialize face matcher service with error handling
	if err := facematch.InitializeFaceMatcherService(); err != nil {
		logger.Error("Failed to initialize face matcher service", logger.LoggerOptions{
			Key:  "error",
			Data: err.Error(),
		})
		// Continue without face matcher for now
	} else {
		logger.Info("Face matcher service initialized successfully")
	}

	sms.InitSMSService()
	payments.InitialisePaymentProcessor()
}

// Used to clean up after services that have been shutdown.
func CleanUpServices() {
	datastore.CleanUp()
	if facematch.GlobalFaceMatcher != nil {
		facematch.GlobalFaceMatcher.Close()
	}
}
