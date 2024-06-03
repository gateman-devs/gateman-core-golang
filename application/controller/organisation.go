package controller

import (
	"net/http"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	org_usecases "authone.usepolymer.co/application/usecases/organisation"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/ipresolver"
	"authone.usepolymer.co/infrastructure/logger"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func CreateOrganisation(ctx *interfaces.ApplicationContext[dto.CreateOrgDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	err := org_usecases.CreateOrgUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, ctx.DeviceID)
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
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", app, nil, nil, ctx.DeviceID)
}
