package application_usecase

import (
	"context"
	"fmt"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	polymercore "authone.usepolymer.co/application/services/polymer-core"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/logger"
)

func CreateApplicationUseCase(ctx any, payload *dto.ApplicationDTO, deviceID *string, userID string, workspaceID string, email string) (*entities.Application, *string) {
	appRepo := repository.ApplicationRepo()
	currentApps, err := appRepo.CountDocs(map[string]interface{}{
		"workspaceID": workspaceID,
	})
	if err != nil {
		logger.Error("an error occured while checking the number of applications a workspace has", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: *payload,
		})
		return nil, nil
	}
	if currentApps >= 20 {
		apperrors.ClientError(ctx, fmt.Sprintf("you have reached the maximum number of applications a workspace can have. Contact %s to assist in creating more.", constants.SUPPORT_EMAIL), nil, nil, deviceID)
		return nil, nil
	}
	appID := utils.GenerateUULDString()
	apiKey, _ := cryptography.EncryptData([]byte(utils.GenerateUULDString()), nil)
	hashedAPIKey, _ := cryptography.CryptoHahser.HashString(string(*apiKey), nil)
	appPriKey := utils.GenerateUULDString()
	app, err := appRepo.CreateOne(context.TODO(), entities.Application{
		ID:                    appPriKey,
		Name:                  payload.Name,
		CreatorID:             userID,
		WorkspaceID:           workspaceID,
		AppImg:                fmt.Sprintf("%s/%s", workspaceID, appPriKey),
		Description:           payload.Description,
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
