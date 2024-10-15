package application_usecase

import (
	"errors"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/ipresolver"
	"authone.usepolymer.co/infrastructure/logger"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FetchAppUseCase(ctx any, appID string, deviceID *string, ip string) (*entities.Application, error) {
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": appID,
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
		apperrors.UnknownError(ctx, err, deviceID)
		return nil, err
	}
	if app == nil {
		apperrors.NotFoundError(ctx, "This application was not found. Seems the link you used might be damaged or malformed. Contact the App owner to report or help you resolve this issue", deviceID)
		return nil, errors.New("app does not exist")
	}
	if app.LocaleRestriction != nil {
		ipData, err := ipresolver.IPResolverInstance.LookUp(ip)
		if err != nil {
			logger.Error("an error occured while resolving users ip for locale restriction", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			apperrors.UnknownError(ctx, err, deviceID)
			return nil, err
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
			apperrors.ClientError(ctx, "Seems you are not in a location that supports this app. If you are using a VPN please turn it off before attempting to access this app.", nil, nil, deviceID)
			return nil, errors.New("invalid location")
		}
	}
	return app, nil
}
