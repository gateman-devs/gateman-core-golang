package application_usecase

import (
	"context"
	"encoding/json"
	"fmt"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/logger"
	messagequeue "authone.usepolymer.co/infrastructure/message_queue"
	queue_tasks "authone.usepolymer.co/infrastructure/message_queue/tasks"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
)

func CreateApplicationUseCase(ctx any, payload *dto.ApplicationDTO, deviceID string, userID string, workspaceID string, email string) (*entities.Application, *string, *string) {
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
		return nil, nil, nil
	}
	if currentApps >= 30 {
		apperrors.ClientError(ctx, fmt.Sprintf("You have reached the maximum number of applications a workspace can have. Contact %s to assist in creating more.", constants.SUPPORT_EMAIL), nil, nil)
		return nil, nil, nil
	}
	apiKey, _ := cryptography.EncryptData([]byte(utils.GenerateUULDString()), nil)
	hashedAPIKey, _ := cryptography.CryptoHahser.HashString(string(*apiKey), nil)
	appSigningKey := utils.GenerateUULDString()
	appSigningKey = fmt.Sprintf("%s%s", appSigningKey, "-g8man")
	encryptedAppSigningKey, _ := cryptography.EncryptData([]byte(appSigningKey), nil)
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
		AppSigningKey:         *encryptedAppSigningKey,
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
		return nil, nil, nil
	}

	emailPayload, err := json.Marshal(queue_tasks.EmailPayload{
		Opts: map[string]any{
			"APP_NAME": payload.Name,
		},
		To:       email,
		Subject:  "Your app has been added to Gateman",
		Template: "application_created",
	})
	if err != nil {
		logger.Error("error marshalling payload for email queue")
		apperrors.FatalServerError(ctx, err)
		return nil, nil, nil
	}
	messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
		Payload:   emailPayload,
		Name:      queue_tasks.HandleEmailDeliveryTaskName,
		Priority:  "high",
		ProcessIn: 1,
	})
	return app, apiKey, utils.GetStringPointer(string(appSigningKey))
}
