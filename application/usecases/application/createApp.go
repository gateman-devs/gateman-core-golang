package application_usecase

import (
	"context"
	"encoding/json"
	"fmt"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/constants"
	"gateman.io/application/controller/dto"
	"gateman.io/application/repository"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
)

func CreateApplicationUseCase(ctx any, payload *dto.ApplicationDTO, deviceID string, userID string, workspaceID string, email string) (*entities.Application, *string, *string, *string, *string, *string) {
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
		return nil, nil, nil, nil, nil, nil
	}
	if currentApps >= 30 {
		apperrors.ClientError(ctx, fmt.Sprintf("You have reached the maximum number of applications a workspace can have. Contact %s to assist in creating more.", constants.SUPPORT_EMAIL), nil, nil, deviceID)
		return nil, nil, nil, nil, nil, nil
	}
	apiKey, _ := utils.GenerateRandomHexKey(32)
	hashedAPIKey, _ := cryptography.CryptoHahser.HashString(apiKey, nil)
	sandboxAPIKey, _ := utils.GenerateRandomHexKey(32)
	hashedSandboxAPIKey, _ := cryptography.CryptoHahser.HashString(sandboxAPIKey, nil)
	appSigningKey, _ := utils.GenerateRandomHexKey(32)
	encryptedAppSigningKey, _ := cryptography.EncryptData([]byte(appSigningKey), nil)
	sandboxAppSigningKey, _ := utils.GenerateRandomHexKey(32)
	encryptedSandboxAppSigningKey, _ := cryptography.EncryptData([]byte(sandboxAppSigningKey), nil)
	appPriKey := utils.GenerateUULDString()
	appID := utils.GenerateUULDString()
	app, err := appRepo.CreateOne(context.TODO(), entities.Application{
		ID:                     appPriKey,
		Name:                   payload.Name,
		Email:                  email,
		CreatorID:              userID,
		AppID:                  appID,
		WorkspaceID:            workspaceID,
		AppImg:                 fmt.Sprintf("%s/%s", workspaceID, appPriKey),
		Description:            payload.Description,
		LocaleRestriction:      payload.LocaleRestriction,
		Verifications:          payload.Verifications,
		RequestedFields:        payload.RequestedFields,
		AppSigningKey:          *encryptedAppSigningKey,
		SandboxAppSigningKey:   *encryptedSandboxAppSigningKey,
		RefreshTokenTTL:        60 * 60 * 24 * 7, // 7 days
		AccessTokenTTL:         60 * 60 * 2,      // 2 hours
		SandboxRefreshTokenTTL: 60 * 60 * 24 * 7, // 7 days
		SandboxAccessTokenTTL:  60 * 60 * 2,      // 2 hours
		SandboxAPIKey:          string(hashedSandboxAPIKey),
		APIKey:                 string(hashedAPIKey),
		CustomFields:           payload.CustomFormFields,
	})
	if err != nil {
		logger.Error("an error occured while creating application", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: *payload,
		})
		return nil, nil, nil, nil, nil, nil
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
		apperrors.FatalServerError(ctx, err, deviceID)
		return nil, nil, nil, nil, nil, nil
	}
	messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
		Payload:   emailPayload,
		Name:      queue_tasks.HandleEmailDeliveryTaskName,
		Priority:  "high",
		ProcessIn: 1,
	})
	return app, &apiKey, &appID, utils.GetStringPointer(string(appSigningKey)), &sandboxAPIKey, &sandboxAppSigningKey
}
