package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
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
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
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
	app, apiKey, appSigningKey := application_usecase.CreateApplicationUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("Email"))
	if app == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "app created", map[string]any{
		"app":           app,
		"apiKey":        apiKey,
		"appSigningKey": appSigningKey,
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
		apperrors.NotFoundError(ctx.Ctx, "this resource does not exist")
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app deleted", nil, nil, nil, nil, nil)
}

func UpdateApplication(ctx *interfaces.ApplicationContext[dto.ApplicationDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	appRepo := repository.ApplicationRepo()
	_, err := appRepo.UpdatePartialByID(ctx.GetStringParameter("id"), ctx.Body)
	if err != nil {
		logger.Error("an error occured while updating application", logger.LoggerOptions{
			Key: "params", Data: ctx.Param,
		}, logger.LoggerOptions{
			Key: "payload", Data: ctx.Body,
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

func ApplicationSignUp(ctx *interfaces.ApplicationContext[dto.ApplicationSignUpDTO]) {
	app, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Body.AppID, ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	appUserRepo := repository.AppUserRepo()
	appUserExists, _ := appUserRepo.CountDocs(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
		"appID":  ctx.Body.AppID,
	})
	if appUserExists != 0 {
		apperrors.ClientError(ctx.Ctx, "Seems you have already signed up for this app", nil, nil)
		return
	}
	userRepo := repository.UserRepo()
	user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
	if user == nil {
		apperrors.NotFoundError(ctx.Ctx, "This user was not found")
		return
	}
	eligible := true
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
			AppID:  ctx.Body.AppID,
			UserID: ctx.GetStringContextData("UserID"),
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
		"appID": ctx.Body.AppID,
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
