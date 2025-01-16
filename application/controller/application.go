package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	application_usecase "authone.usepolymer.co/application/usecases/application"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/logger"
	messagequeue "authone.usepolymer.co/infrastructure/message_queue"
	queue_tasks "authone.usepolymer.co/infrastructure/message_queue/tasks"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateApplication(ctx *interfaces.ApplicationContext[dto.ApplicationDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	if len(ctx.Body.RequestedFields) == 0 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("requestedFields cannot be empty")})
		return
	}
	for _, field := range *ctx.Body.RequiredVerifications {
		if !utils.HasItemString(&constants.AVAILABLE_REQUIRED_DATA_POINTS, field) {
			apperrors.ValidationFailedError(ctx.Ctx, &[]error{fmt.Errorf("%s is not allowed in requested field", field)})
			return
		}
	}
	if ctx.Body.LocaleRestriction != nil {
		for _, r := range *ctx.Body.LocaleRestriction {
			valiedationErr := validator.ValidatorInstance.ValidateStruct(r)
			if valiedationErr != nil {
				apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
				return
			}
		}
	}
	app, apiKey, appSigningKey, sandboxAPIKey, sandboxAppSigningKey := application_usecase.CreateApplicationUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("Email"))
	if app == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "app created", map[string]any{
		"app":                  app,
		"apiKey":               apiKey,
		"appSigningKey":        appSigningKey,
		"sandboxAPIKey":        sandboxAPIKey,
		"sandboxAppSigningKey": sandboxAppSigningKey,
	}, nil, nil, nil, nil)
}

func FetchAppCreationConfigInfo(ctx *interfaces.ApplicationContext[any]) {
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "required fields", map[string]any{
		"requiredFields": constants.AVAILABLE_REQUIRED_DATA_POINTS,
	}, nil, nil, nil, nil)
}

func FetchAppDetails(ctx *interfaces.ApplicationContext[any]) {
	app, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Param["id"].(string), ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "app fetched", app, nil, nil, nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "apps fetched", apps, nil, nil, nil, nil)
}

func DeleteApplication(ctx *interfaces.ApplicationContext[any]) {
	appRepo := repository.ApplicationRepo()
	deleted, err := appRepo.DeleteByID(ctx.GetStringParameter("id"))
	if err != nil {
		logger.Error("an error occured while trying to fetch workspace apps", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if deleted == 0 {
		apperrors.NotFoundError(ctx.Ctx, "this application does not exist")
		return
	}

	deleteAppPayload, err := json.Marshal(queue_tasks.DeleteAppPayload{
		ID:          ctx.GetStringParameter("id"),
		WorkspaceID: *ctx.GetHeader("X-Workspace-Id"),
	})
	if err != nil {
		logger.Error("error marshalling payload for delete app queue")
		apperrors.FatalServerError(ctx, err)
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
		ProcessIn: uint(processIn),
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app deleted", nil, nil, nil, nil, nil)
}

func UpdateApplication(ctx *interfaces.ApplicationContext[dto.UpdateApplications]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
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
	if ctx.Body.RequiredVerifications != nil {
		payload["requiredVerifications"] = ctx.Body.RequiredVerifications
	}
	appRepo := repository.ApplicationRepo()
	_, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), payload)
	if err != nil {
		logger.Error("an error occured while updating application", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: payload,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app updated", nil, nil, nil, nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found")
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "API key updated. This will only be displayed once", apiKey, nil, nil, nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found")
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "App Signing Key updated. This will only be displayed once", appSigningKey, nil, nil, nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found")
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Sandbox API key updated. This will only be displayed once", apiKey, nil, nil, nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if app == 0 {
		apperrors.NotFoundError(ctx.Ctx, "Invalid app id provided. App not found")
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Sandbox App Signing Key updated. This will only be displayed once", appSigningKey, nil, nil, nil, nil)
}

func UpdateAccessRefreshTokenTTL(ctx *interfaces.ApplicationContext[dto.UpdateAccessRefreshTokenTTL]) {
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
	fmt.Println(updateFields)
	appRepo := repository.ApplicationRepo()
	appRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id":       ctx.GetStringParameter("id"),
		"workspaceID": *ctx.GetHeader("X-Workspace-Id"),
	}, updateFields)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "TTL updated", nil, nil, nil, nil, nil)
}

func ApplicationSignUp(ctx *interfaces.ApplicationContext[dto.ApplicationSignUpDTO]) {
	app, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Body.AppID, ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	userRepo := repository.UserRepo()
	user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
	if user == nil {
		apperrors.NotFoundError(ctx.Ctx, "This user was not found")
		return
	}
	appUserRepo := repository.AppUserRepo()
	appUserExists, _ := appUserRepo.FindOneByFilter(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
		"appID":  ctx.Body.AppID,
	})
	eligible := true
	if appUserExists != nil {
		requestedFields := map[string]any{}
		userValue := reflect.ValueOf(*user)

		for _, field := range app.RequestedFields {
			userField := userValue.FieldByName(field.Name)
			if !userField.IsValid() {
				eligible = false
				continue
			}
			var userFieldData entities.KYCData[any]
			actualValue := userField.Interface()
			jsonBytes, _ := json.Marshal(actualValue)
			json.Unmarshal(jsonBytes, &userFieldData)

			// If Verified field doesn't exist or is not true, add to results
			if userFieldData.Value == nil || !userFieldData.Verified {
				eligible = false
			}
			requestedFields[field.Name] = userFieldData.Value
		}
		return
	}
	outstandingIDs := []string{}
	for _, id := range *app.RequiredVerifications {
		if id == "nin" {
			if user.NIN == nil {
				outstandingIDs = append(outstandingIDs, "nin")
				eligible = false
			}
		} else {
			if user.BVN == nil {
				outstandingIDs = append(outstandingIDs, "bvn")
				eligible = false
			}
		}
	}
	var results []string
	requestedFields := map[string]any{}
	userValue := reflect.ValueOf(*user)

	for _, field := range app.RequestedFields {
		userField := userValue.FieldByName(field.Name)
		if !userField.IsValid() {
			results = append(results, field.Name)
			eligible = false
			continue
		}
		var userFieldData entities.KYCData[any]
		actualValue := userField.Interface()
		jsonBytes, _ := json.Marshal(actualValue)
		json.Unmarshal(jsonBytes, &userFieldData)

		// If Verified field doesn't exist or is not true, add to results
		if userFieldData.Value == nil || !userFieldData.Verified {
			results = append(results, field.Name)
			eligible = false
		}
		requestedFields[field.Name] = userFieldData.Value
	}

	payload := map[string]any{}
	var msg string
	if eligible {
		msg = "Sign up successful"
	} else {
		msg = "Additional info is required to sign up to this app"
		payload["missingIDs"] = outstandingIDs
		payload["unverifiedFields"] = results
	}
	if eligible {
		appUserRepo.CreateOne(context.TODO(), entities.AppUser{
			AppID:       ctx.Body.AppID,
			UserID:      ctx.GetStringContextData("UserID"),
			WorkspaceID: app.WorkspaceID,
		})
		decryptedAppSigningKey, _ := cryptography.DecryptData(app.AppSigningKey, nil)
		token, _ := auth.GenerateAppUserToken(auth.ClaimsData{
			IssuedAt:  time.Now().Unix(),
			ExpiresAt: time.Now().Add(time.Minute * 30).Unix(),
			Payload:   requestedFields,
		}, string(decryptedAppSigningKey))
		payload["token"] = token
		payload["expiresAt"] = time.Now().Add(time.Minute * 30)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, msg, payload, nil, nil, nil, nil)
}

func FetchAppUsers(ctx *interfaces.ApplicationContext[dto.FetchAppUsersDTO]) {
	filter := map[string]interface{}{
		"appID":       ctx.Body.AppID,
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users fetched", users, nil, nil, nil, nil)
}

func BlockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
	}, map[string]any{
		"blocked": true,
	})
	if err != nil {
		logger.Error("an error occured while blocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users blocked", nil, nil, nil, nil, nil)
}

func UnblockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
	}, map[string]any{
		"blocked": false,
	})
	if err != nil {
		logger.Error("an error occured while unblocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users unblocked", nil, nil, nil, nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "apps fetched", apps, nil, nil, nil, nil)
}

func GetAppMetrics(ctx *interfaces.ApplicationContext[dto.FetchAppMetrics]) {
	appMetrics := map[string]any{}
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindByID(ctx.Body.ID, options.FindOne().SetProjection(map[string]any{
		"name":      1,
		"createdAt": 1,
		"appImg":    1,
	}))
	appMetrics["name"] = app.Name
	appMetrics["createdAt"] = app.CreatedAt
	appMetrics["appImg"] = app.AppImg
	appUserRepo := repository.AppUserRepo()
	usersCount, _ := appUserRepo.CountDocs(map[string]any{
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
		"appID":       ctx.Body.ID,
	})
	appMetrics["usersCount"] = usersCount
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "metrics fetched", appMetrics, nil, nil, nil, nil)
}
