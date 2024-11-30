package controller

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	application_usecase "authone.usepolymer.co/application/usecases/application"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
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
	app, apiKey := application_usecase.CreateApplicationUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("Email"))
	if app == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "app created", map[string]any{
		"app":    app,
		"apiKey": apiKey,
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
	_, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Body.AppID, ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	appUserRepo := repository.AppUserRepo()
	context, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	appUserRepo.CreateOne(context, entities.AppUser{
		AppID:  ctx.Body.AppID,
		UserID: ctx.GetStringContextData("UserID"),
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "application signup", nil, nil, nil, nil, nil)
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
