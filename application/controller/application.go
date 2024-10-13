package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	application_usecase "authone.usepolymer.co/application/usecases/application"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/ipresolver"
	"authone.usepolymer.co/infrastructure/logger"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
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
	for _, field := range *ctx.Body.RequiredVerifications {
		if !utils.HasItemString(&constants.AVAILABLE_REQUIRED_DATA_POINTS, field) {
			apperrors.ValidationFailedError(ctx.Ctx, &[]error{fmt.Errorf("%s is not allowed in requested field", field)}, ctx.DeviceID)
			return
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
	app, apiKey := application_usecase.CreateApplicationUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("Email"))
	if app == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", map[string]any{
		"app":    app,
		"apiKey": apiKey,
	}, nil, nil, ctx.DeviceID)
}

func FetchAppCreationConfigInfo(ctx *interfaces.ApplicationContext[any]) {
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "required fields", map[string]any{
		"requiredFields": constants.AVAILABLE_REQUIRED_DATA_POINTS,
	}, nil, nil, ctx.DeviceID)
}

func FetchAppDetails(ctx *interfaces.ApplicationContext[any]) {
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": ctx.Param["id"],
	}, options.FindOne().SetProjection(map[string]any{
		"name":                  1,
		"requiredVerifications": 1,
		"requestedFields":       1,
		"localeRestriction":     1,
		"description":           1,
	}))
	if err != nil {
		logger.Error("an error occured while fethcing app details", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if app == nil {
		apperrors.NotFoundError(ctx.Ctx, "This application was not found. Seems the link you used might be damaged or malformed. Contact the App owner to report or help you resolve this issue", ctx.DeviceID)
		return
	}
	if app.LocaleRestriction != nil {
		ipData, err := ipresolver.IPResolverInstance.LookUp(ctx.Keys["ip"].(string))
		if err != nil {
			logger.Error("an error occured while resolving users ip for locale restriction", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
			return
		}
		passed := false
		for _, locale := range *app.LocaleRestriction {
			if locale.States != nil {
				if utils.HasItemString(locale.States, strings.ToLower(ipData.City)) && locale.Country == ipData.CountryCode {
					passed = true
					break
				}
			} else {
				if locale.Country == ipData.CountryCode {
					passed = true
					break
				}
			}
		}
		if !passed {
			apperrors.ClientError(ctx.Ctx, "Seems you are not in a location that supports this app. If you are using a VPN please turn it off before attempting to access this app.", nil, nil, ctx.DeviceID)
			return
		}
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org fetched", app, nil, nil, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "apps fetched", apps, nil, nil, ctx.DeviceID)
}

func DeleteApplication(ctx *interfaces.ApplicationContext[any]) {
	appRepo := repository.ApplicationRepo()
	deleted, err := appRepo.DeleteByID(ctx.GetStringParameter("id"))
	if err != nil {
		logger.Error("an error occured while trying to fetch workspace apps", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if deleted == 0 {
		apperrors.NotFoundError(ctx.Ctx, "this resource does not exist", ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "apps fetched", nil, nil, nil, ctx.DeviceID)
}

func UpdateApplication(ctx *interfaces.ApplicationContext[dto.ApplicationDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app updated", nil, nil, nil, ctx.DeviceID)
}
