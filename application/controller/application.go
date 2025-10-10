package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/constants"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	services "gateman.io/application/services/application"
	application_usecase "gateman.io/application/usecases/application"
	auth_usecases "gateman.io/application/usecases/auth"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/cryptography"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/totp"
	"gateman.io/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateApplication(ctx *interfaces.ApplicationContext[dto.ApplicationDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if len(ctx.Body.RequestedFields) == 0 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("requestedFields cannot be empty")}, ctx.DeviceID)
		return
	}
	if ctx.Body.Verifications != nil {
		for _, field := range *ctx.Body.Verifications {
			if !utils.HasItemString(&constants.AVAILABLE_REQUIRED_DATA_POINTS, field.Name) {
				apperrors.ValidationFailedError(ctx.Ctx, &[]error{fmt.Errorf("%s is not allowed in requested field", field.Name)}, ctx.DeviceID)
				return
			}
		}
	}
	if ctx.Body.LocaleRestriction != nil {
		for _, r := range *ctx.Body.LocaleRestriction {
			valiedationErr := validator.ValidatorInstance.ValidateStruct(r)
			if valiedationErr != nil {
				apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
				return
			}
		}
	}
	if ctx.Body.CustomFormFields != nil && len(*ctx.Body.CustomFormFields) > 48 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("duplicate validation options detected on custom form fields")}, ctx.DeviceID)
		return
	}
	if len(ctx.Body.RequestedFields) > 11 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("requested fields cannot contain more than 11 items")}, ctx.DeviceID)
		return
	}
	if ctx.Body.Verifications != nil && len(*ctx.Body.Verifications) > 4 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("verifications cannot contain more than 4 items")}, ctx.DeviceID)
		return
	}
	if ctx.Body.LocaleRestriction != nil && len(*ctx.Body.LocaleRestriction) > 300 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("locale restrictions cannot contain more than 300 items")}, ctx.DeviceID)
		return
	}
	app, apiKey, appID, appSigningKey, sandboxAPIKey, sandboxAppSigningKey := application_usecase.CreateApplicationUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("Email"))
	if app == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "app created", map[string]any{
		"url":                  fmt.Sprintf("%s/app/authenticate/%s", os.Getenv("CLIENT_URL"), app.ID),
		"apiKey":               apiKey,
		"appID":                appID,
		"appSigningKey":        appSigningKey,
		"sandboxAPIKey":        sandboxAPIKey,
		"sandboxAppSigningKey": sandboxAppSigningKey,
	}, nil, nil, &ctx.DeviceID)
}

func FetchAppCreationConfigInfo(ctx *interfaces.ApplicationContext[any]) {
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "config info fetched", map[string]any{
		"requiredFields":    constants.AVAILABLE_REQUIRED_DATA_POINTS,
		"customFieldTypes":  constants.CUSTOM_FIELD_TYPES,
		"validationOptions": entities.ValidationRules,
	}, nil, nil, &ctx.DeviceID)
}

func FetchAppDetails(ctx *interfaces.ApplicationContext[any]) {
	app, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Param["id"].(string), ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	var isSignedIn bool
	isUserSignedIn := auth_usecases.IsUserSignedIn(ctx.Ctx, ctx.Keys["accessToken"], nil, ctx.DeviceID)
	isSignedIn = isUserSignedIn.IsAuthenticated
	var signUpStatus map[string]any
	if isSignedIn {
		userRepo := repository.UserRepo()
		user, _ := userRepo.FindByID(isUserSignedIn.UserID)
		_, _, signUpStatus, _ = services.ProcessUserSignUp(app, user, ctx.Keys["ip"].(string))
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app fetched", map[string]any{
		"app":          app,
		"isSignedIn":   isSignedIn,
		"signUpStatus": signUpStatus,
	}, nil, nil, &ctx.DeviceID)
}

func FetchWorkspaceApps(ctx *interfaces.ApplicationContext[any]) {
	appRepo := repository.ApplicationRepo()
	apps, err := appRepo.FindMany(map[string]interface{}{
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	})
	if err != nil {
		logger.Error("an error occured while trying to fetch workspace apps", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
	}
	if apps != nil {
		for i, app := range *apps {
			url, _ := fileupload.FileUploader.GeneratedSignedURL(app.AppImg, types.SignedURLPermission{
				Read: true,
			}, time.Hour*1)
			app.AppImg = *url
			(*apps)[i] = app
		}
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "apps fetched", apps, nil, nil, &ctx.DeviceID)
}

func DeleteApplication(ctx *interfaces.ApplicationContext[any]) {
	appRepo := repository.ApplicationRepo()
	deleted, err := appRepo.FindByID(ctx.GetStringParameter("id"))
	if err != nil {
		logger.Error("an error occured while trying to fetch workspace apps", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if deleted == nil {
		apperrors.NotFoundError(ctx.Ctx, "this application does not exist", &ctx.DeviceID)
		return
	}
	appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"disabled": true,
	})
	deleteAppPayload, err := json.Marshal(queue_tasks.DeleteAppPayload{
		ID:          ctx.GetStringParameter("id"),
		WorkspaceID: ctx.GetStringContextData("WorkspaceID"),
	})
	if err != nil {
		logger.Error("error marshalling payload for delete app queue")
		apperrors.FatalServerError(ctx, err, ctx.DeviceID)
		return
	}
	deleteIn := os.Getenv("DELETE_APP_IN_SECONDS")
	if deleteIn == "" {
		deleteIn = "1"
	}
	processIn, _ := strconv.Atoi(deleteIn)
	messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
		Payload:   deleteAppPayload,
		Name:      queue_tasks.HandleAppDeletionTaskName,
		Priority:  mq_types.Low,
		ProcessIn: time.Duration(processIn),
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app deleted", nil, nil, nil, &ctx.DeviceID)
}

func UpdateApplication(ctx *interfaces.ApplicationContext[dto.UpdateApplications]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	payload := map[string]any{}
	if ctx.Body.Name != nil {
		payload["name"] = ctx.Body.Name
	}
	if ctx.Body.Description != nil {
		payload["description"] = ctx.Body.Description
	}
	if ctx.Body.LocaleRestriction != nil {
		payload["localeRestriction"] = ctx.Body.LocaleRestriction
	}
	if ctx.Body.Verifications != nil {
		payload["verifications"] = ctx.Body.Verifications
	}
	if ctx.Body.RequestedFields != nil {
		payload["requestedFields"] = ctx.Body.RequestedFields
	}
	if ctx.Body.CustomFormFields != nil {
		payload["customFields"] = ctx.Body.CustomFormFields
	}
	if ctx.Body.PaymentCard != nil {
		workspaceRepo := repository.WorkspaceRepository()
		workspace, err := workspaceRepo.FindByID(ctx.GetStringContextData("WorkspaceID"))
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		var card *entities.CardInfo
		for _, savedCard := range workspace.PaymentDetails {
			if savedCard.ID == *ctx.Body.PaymentCard {
				card = &savedCard
				break
			}
		}
		if card == nil {
			apperrors.ClientError(ctx.Ctx, "Saved card not found", nil, nil, ctx.DeviceID)
			return
		}
		payload["paymentCard"] = ctx.Body.PaymentCard
	}
	if ctx.Body.Interval != nil {
		payload["interval"] = ctx.Body.Interval
	}
	if ctx.Body.SubscriptionID != nil {
		subscriptionRepo := repository.SubscriptionPlanRepo()
		sub, _ := subscriptionRepo.FindByID(*ctx.Body.SubscriptionID)
		if sub == nil {
			apperrors.ClientError(ctx.Ctx, "invalid subscription id", nil, nil, ctx.DeviceID)
			return
		}
		payload["subscriptionID"] = ctx.Body.SubscriptionID
	}
	appRepo := repository.ApplicationRepo()
	_, err := appRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id":         ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}, payload)
	if err != nil {
		logger.Error("an error occured while updating application", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: payload,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app updated", nil, nil, nil, &ctx.DeviceID)
}

func RefreshAppAPIKey(ctx *interfaces.ApplicationContext[any]) {
	apiKey, _ := cryptography.EncryptData([]byte(utils.GenerateUULDString()), nil)
	hashedAPIKey, _ := cryptography.CryptoHahser.HashString(string(*apiKey), nil)
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"apiKey": string(hashedAPIKey),
	})
	if err != nil {
		logger.Error("an error occured while updating api key", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "API key updated. This will only be displayed once", apiKey, nil, nil, &ctx.DeviceID)
}

func UpdateWhiteListedIPs(ctx *interfaces.ApplicationContext[dto.UpdateWhitelistIPDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	invalidIP := utils.ValidateIPAddresses(ctx.Body.IPs)
	if invalidIP {
		apperrors.ClientError(ctx.Ctx, "please enter only valid IP addresses", nil, nil, ctx.DeviceID)
		return
	}
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"whiteListedIPs": utils.MakeStringArrayUnique(ctx.Body.IPs),
	})
	if err != nil {
		logger.Error("an error occured while updating whitelisted ips", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "IP Whitelist updated", nil, nil, nil, &ctx.DeviceID)
}

func RefreshAppSigningKey(ctx *interfaces.ApplicationContext[any]) {
	appSigningKey := utils.GenerateUULDString()
	appSigningKey = fmt.Sprintf("%s%s", appSigningKey, "-g8man")
	encryptedAppSigningKey, _ := cryptography.EncryptData([]byte(appSigningKey), nil)
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"appSigningKey": *encryptedAppSigningKey,
	})
	if err != nil {
		logger.Error("an error occured while updating app signing key", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "App Signing Key updated. This will only be displayed once", appSigningKey, nil, nil, &ctx.DeviceID)
}

func TogglePinProtectionSetting(ctx *interfaces.ApplicationContext[dto.TogglePinProtectionSettingDTO]) {
	activeSubRepo := repository.ActiveSubscriptionRepo()
	activeSub, err := activeSubRepo.FindOneByFilter(map[string]interface{}{
		"appID": ctx.GetStringParameter("id"),
	})
	if err != nil {
		logger.Error("an error occured while fetching active subcription for toggle pin protected account", logger.LoggerOptions{
			Key: "id", Data: ctx.GetStringParameter("id"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if activeSub == nil || activeSub.ActiveSubName == entities.Free {
		apperrors.ClientError(ctx.Ctx, "Your current tier does not support pin protected accounts. Please upgrade to the Gateman Essential plan to get this feature", nil, nil, ctx.DeviceID)
		return
	}
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"pinProtected": ctx.Body.Activated,
	})
	if err != nil {
		logger.Error("an error occured while updating protected pin setting", logger.LoggerOptions{
			Key: "id", Data: ctx.GetStringParameter("id"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Pin protection setting updated", nil, nil, nil, &ctx.DeviceID)
}

func ToggleMFAProtectionSetting(ctx *interfaces.ApplicationContext[dto.ToggleMFAProtectionSettingDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	activeSubRepo := repository.ActiveSubscriptionRepo()
	activeSub, err := activeSubRepo.FindOneByFilter(map[string]interface{}{
		"appID": ctx.Body.ID,
	})
	if err != nil {
		logger.Error("an error occured while fetching active subcription for toggle MFA protected accounts", logger.LoggerOptions{
			Key: "id", Data: ctx.Body.ID,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if activeSub == nil || activeSub.ActiveSubName != entities.Premium {
		apperrors.ClientError(ctx.Ctx, "Your current tier does not support MFA protected accounts. Please upgrade to the Gateman Premium plan to get this feature", nil, nil, ctx.DeviceID)
		return
	}
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.Body.ID, map[string]any{
		"requireAppMFA": ctx.Body.Activated,
	})
	if err != nil {
		logger.Error("an error occured while updating mfa protected setting", logger.LoggerOptions{
			Key: "id", Data: ctx.GetStringParameter("id"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "MFA protection setting updated", nil, nil, nil, &ctx.DeviceID)
}

func RefreshSandboxAppAPIKey(ctx *interfaces.ApplicationContext[any]) {
	apiKey, _ := cryptography.EncryptData([]byte(utils.GenerateUULDString()), nil)
	hashedAPIKey, _ := cryptography.CryptoHahser.HashString(string(*apiKey), nil)
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"sandBoxAPIKey": string(hashedAPIKey),
	})
	if err != nil {
		logger.Error("an error occured while updating sandbox api key", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Sandbox API key updated. This will only be displayed once", apiKey, nil, nil, &ctx.DeviceID)
}

func RefreshSandboxAppSigningKey(ctx *interfaces.ApplicationContext[any]) {
	appSigningKey := utils.GenerateUULDString()
	appSigningKey = fmt.Sprintf("sandbox-%s-g8man", appSigningKey)
	encryptedAppSigningKey, _ := cryptography.EncryptData([]byte(appSigningKey), nil)
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), map[string]any{
		"sandBoxAppSigningKey": *encryptedAppSigningKey,
	})
	if err != nil {
		logger.Error("an error occured while updating app signing key", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Sandbox App Signing Key updated. This will only be displayed once", appSigningKey, nil, nil, &ctx.DeviceID)
}

func UpdateAccessRefreshTokenTTL(ctx *interfaces.ApplicationContext[dto.UpdateAccessRefreshTokenTTL]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	updateFields := map[string]any{}

	if ctx.Body.RefreshTokenTTL != nil {
		updateFields["refreshTokenTTL"] = ctx.Body.RefreshTokenTTL
	}

	if ctx.Body.AccessTokenTTL != nil {
		updateFields["accessTokenTTL"] = ctx.Body.AccessTokenTTL
	}

	if ctx.Body.SandboxRefreshTokenTTL != nil {
		updateFields["sandboxRefreshTokenTTL"] = ctx.Body.SandboxRefreshTokenTTL
	}

	if ctx.Body.SandboxAccessTokenTTL != nil {
		updateFields["sandboxAccessTokenTTL"] = ctx.Body.SandboxAccessTokenTTL
	}
	appRepo := repository.ApplicationRepo()
	updated, _ := appRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id":         ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}, updateFields)
	if !updated {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found", &ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "TTL updated", nil, nil, nil, &ctx.DeviceID)
}

func ApplicationSignUp(ctx *interfaces.ApplicationContext[dto.ApplicationSignUpDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	app, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Body.AppID, ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	userRepo := repository.UserRepo()
	user, err := userRepo.FindByID(ctx.GetStringContextData("UserID"))
	if err != nil {
		logger.Error("an error occured while fetching user for app signup", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if user == nil {
		apperrors.NotFoundError(ctx.Ctx, "This user was not found", &ctx.DeviceID)
		return
	}
	appUserRepo := repository.AppUserRepo()
	appUserExists, _ := appUserRepo.FindOneByFilter(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
		"appID":  ctx.Body.AppID,
	})
	var responseCode *uint
	if appUserExists != nil {
		if appUserExists.Blocked {
			apperrors.AuthenticationError(ctx.Ctx, "Your access to this application has been restricted", ctx.DeviceID)
			return
		}
		if app.PinProtected {
			if appUserExists.Pin == nil {
				responseCode = &constants.SET_APP_PIN
			}
			pinMatch := cryptography.CryptoHahser.VerifyHashData(*appUserExists.Pin, *ctx.Body.Pin)
			if !pinMatch {
				apperrors.AuthenticationError(ctx.Ctx, "Incorrect pin", ctx.DeviceID)
				return
			}
		}
		eligible, msg, payload, requestedFields := services.ProcessUserSignUp(app, user, ctx.Keys["ip"].(string))
		if eligible {
			block, err := services.CheckMonthlyLimit(ctx.Ctx, app.ID, appUserExists.ID, ctx.DeviceID)
			if err != nil || block {
				return
			}
			payload, err := services.GenerateAuthTokens(payload, app, ctx.UserAgent, ctx.DeviceID, ctx.GetStringContextData("UserID"), requestedFields)
			if err != nil {
				apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
				return
			}
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, msg, payload, nil, responseCode, &ctx.DeviceID)
		}
		server_response.Responder.Respond(ctx.Ctx, http.StatusBadRequest, msg, payload, nil, nil, &ctx.DeviceID)
	} else {
		if app.PinProtected && ctx.Body.Pin == nil {
			apperrors.ClientError(ctx.Ctx, "Provide your login pin", nil, nil, ctx.DeviceID)
			return
		}
		eligible, msg, payload, requestedFields := services.ProcessUserSignUp(app, user, ctx.Keys["ip"].(string))
		if eligible {
			var pin []byte
			if ctx.Body.Pin != nil {
				pin, _ = cryptography.CryptoHahser.HashString(*ctx.Body.Pin, nil)
			}
			appUserExists, _ := appUserRepo.CreateOne(context.TODO(), entities.AppUser{
				AppID:       ctx.Body.AppID,
				UserID:      ctx.GetStringContextData("UserID"),
				WorkspaceID: app.WorkspaceID,
				Pin:         utils.GetStringPointer(string(pin)),
			})
			block, err := services.CheckMonthlyLimit(ctx.Ctx, app.ID, appUserExists.ID, ctx.DeviceID)
			if err != nil || block {
				return
			}
			payload, err := services.GenerateAuthTokens(payload, app, ctx.UserAgent, ctx.DeviceID, ctx.GetStringContextData("UserID"), requestedFields)
			if err != nil {
				apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			}
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, msg, payload, nil, nil, &ctx.DeviceID)
			return
		}
		server_response.Responder.Respond(ctx.Ctx, http.StatusBadRequest, msg, payload, nil, nil, &ctx.DeviceID)
	}
}

func SubmitCustomAppForm(ctx *interfaces.ApplicationContext[dto.SubmitCustomAppFormDTO]) {
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": ctx.Body.AppID,
	})
	if err != nil {
		logger.Error("an error occured while trying to fetch application for custom for submititon", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if app == nil {
		apperrors.NotFoundError(ctx.Ctx, "App not found", &ctx.DeviceID)
		return
	}
	if app.CustomFields == nil {
		apperrors.ClientError(ctx.Ctx, fmt.Sprintf("%s does not have a custom form", app.Name), nil, nil, ctx.DeviceID)
		return
	}
	appUserRepo := repository.AppUserRepo()
	appUser, err := appUserRepo.FindOneByFilter(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
		"appID":  ctx.Body.AppID,
	})
	if err != nil {
		logger.Error("an error occured while trying to fetch application user for custom for submititon", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if appUser == nil {
		apperrors.ClientError(ctx.Ctx, "Sign up to the app before attempting to submit the form", nil, nil, ctx.DeviceID)
		return
	}
	var validationErr []error
	for _, customField := range *app.CustomFields {
		if customField.Page != ctx.Body.Page {
			continue
		}
		fieldValue := ctx.Body.Data[customField.DBKey]
		var rulesBuilder strings.Builder

		for i, rule := range customField.Rules {
			customRule := entities.ValidationRules[rule.Name]
			if i > 0 {
				rulesBuilder.WriteString(",")
			}
			rulesBuilder.WriteString(customRule.Tag)
			if rule.Value != nil {
				rulesBuilder.WriteString(fmt.Sprintf("=%s", *rule.Value))
			}
		}

		rules := rulesBuilder.String()
		if err := validator.ValidatorInstance.ValidateValue(fieldValue, rules); err != nil {
			validationErr = append(validationErr, errors.New(strings.Replace(err.Error(), "Field", customField.Name, 1)))
		}
	}
	if len(validationErr) != 0 {
		apperrors.ValidationFailedError(ctx.Ctx, &validationErr, ctx.DeviceID)
		return
	}
	appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
		"appID":  ctx.Body.AppID}, map[string]any{
		"customFieldData": ctx.Body.Data,
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "form submitted", nil, nil, nil, &ctx.DeviceID)
}

func FetchAppUsers(ctx *interfaces.ApplicationContext[dto.FetchAppUsersDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	filter := map[string]interface{}{
		"appID":       ctx.Body.AppID,
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}
	if ctx.Body.Blocked != nil {
		filter["blocked"] = ctx.Body.Blocked
	}
	if ctx.Body.Deleted != nil {
		filter["deletedAt"] = map[string]any{"$ne": nil}
	}
	appUserRepo := repository.AppUserRepo()
	users, err := appUserRepo.FindManyPaginated(filter, ctx.Body.PageSize, ctx.Body.LastID, int(ctx.Body.Sort))
	if err != nil {
		logger.Error("an error occured while fetching apps users", logger.LoggerOptions{
			Key:  "appID",
			Data: ctx.Body.AppID,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users fetched", users, nil, nil, &ctx.DeviceID)
}

func BlockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}, map[string]any{
		"blocked":       true,
		"blockedUserAt": time.Now(),
		"blockedReason": ctx.Body.Reason,
	})
	if err != nil {
		logger.Error("an error occured while blocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users blocked", nil, nil, nil, &ctx.DeviceID)
}

func UnblockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}, map[string]any{
		"blocked":       false,
		"blockedUserAt": nil,
		"blockedReason": nil,
	})
	if err != nil {
		logger.Error("an error occured while unblocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users unblocked", nil, nil, nil, &ctx.DeviceID)
}

func FetchUserApps(ctx *interfaces.ApplicationContext[any]) {
	appUserRepo := repository.AppUserRepo()
	apps, err := appUserRepo.FindMany(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
	})
	if err != nil {
		logger.Error("an error occured while fetching user apps", logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "apps fetched", apps, nil, nil, &ctx.DeviceID)
}

func GetAppMetrics(ctx *interfaces.ApplicationContext[dto.FetchAppMetrics]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	appMetrics := map[string]any{}
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindByID(ctx.Body.ID, options.FindOne().SetProjection(map[string]any{
		"name":      1,
		"createdAt": 1,
		"disabled":  1,
		"appID":     1,
	}))
	if app == nil {
		apperrors.NotFoundError(ctx.Ctx, "App not found", &ctx.DeviceID)
		return
	}
	if app.Disabled {
		apperrors.ClientError(ctx.Ctx, "This app has been deactivated", nil, nil, ctx.DeviceID)
		return
	}
	appMetrics["name"] = app.Name
	appMetrics["createdAt"] = app.CreatedAt
	appUserRepo := repository.AppUserRepo()
	usersCount, _ := appUserRepo.CountDocs(map[string]any{
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		"appID":       app.AppID,
	})
	appMetrics["usersCount"] = usersCount
	appMetrics["disabled"] = app.Disabled
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "metrics fetched", appMetrics, nil, nil, &ctx.DeviceID)
}

func SetUpMFA(ctx *interfaces.ApplicationContext[any]) {
	appUserRepo := repository.AppUserRepo()
	user, _ := appUserRepo.FindByID(ctx.GetStringContextData("UserID"))
	if user == nil {
		apperrors.NotFoundError(ctx.Ctx, "User not found", &ctx.DeviceID)
		return
	}
	if user.AuthenticatorSecret != nil {
		apperrors.ClientError(ctx.Ctx, "MFA has already been set up on this account", nil, nil, ctx.DeviceID)
		return
	}
	_, url, err := totp.TOTPService.GenerateSecret(ctx.GetStringContextData("UserID"))
	if err != nil {
		logger.Error("an error occured while trying to generate TOTP code", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "mfa secret generated", url, nil, nil, &ctx.DeviceID)
}
