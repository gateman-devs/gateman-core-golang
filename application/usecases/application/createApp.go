package application_usecase

import (
	"context"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	polymercore "authone.usepolymer.co/application/services/polymer-core"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/logger"
)

func CreateApplicationUseCase(ctx any, payload *dto.ApplicationDTO, deviceID *string, userID string, orgID string, email string) (*entities.Application, *string) {
	appID := utils.GenerateUULDString()
	apiKey, _ := cryptography.EncryptData([]byte(utils.GenerateUULDString()), nil)
	hashedAPIKey, _ := cryptography.CryptoHahser.HashString(string(*apiKey))
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.CreateOne(context.TODO(), entities.Application{
		Name:                  payload.Name,
		CreatorID:             userID,
		OrgID:                 orgID,
		LocaleRestriction:     payload.LocaleRestriction,
		RequiredVerifications: payload.RequiredVerifications,
		RequestedFields:       payload.RequestedFields,
		AppID:                 appID,
		APIKey:                string(hashedAPIKey),
	})
	if err != nil {
		logger.Error("an error occured while creating application", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: *payload,
		})
		return nil, nil
	}
	err = polymercore.PolymerService.SendEmail("application_created", email, "Your new application has been created!", map[string]any{
		"APP_NAME": payload.Name,
	})
	if err != nil {
		apperrors.UnknownError(ctx, err, deviceID)
		return nil, nil
	}
	return app, apiKey
}
