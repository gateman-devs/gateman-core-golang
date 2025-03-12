package middlewares

import (
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/infrastructure/cryptography"
)

func AppAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], ipAddress string) (*interfaces.ApplicationContext[any], bool) {
	apiKeyPointer := ctx.GetHeader("X-Api-Key")
	if apiKeyPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an api key", ctx.DeviceID)
		return nil, false
	}
	apiKey := *apiKeyPointer
	appIDPointer := ctx.GetHeader("X-App-Id")
	if appIDPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an app id", ctx.DeviceID)
		return nil, false
	}
	appID := *appIDPointer
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": appID,
	})
	if app == nil {
		apperrors.NotFoundError(ctx.Ctx, "invalid credentials", ctx.GetHeader("X-Device-Id"))
		return nil, false
	}
	var appAPIKey string
	if os.Getenv("ENV") != "production" {
		appAPIKey = app.SandboxAPIKey
	} else {
		appAPIKey = app.APIKey
	}
	match := cryptography.CryptoHahser.VerifyHashData(appAPIKey, apiKey)
	if !match {
		apperrors.ClientError(ctx.Ctx, "invalid credentials", nil, nil, ctx.DeviceID)
		return nil, false
	}
	if os.Getenv("ENV") == "production" && app.WhiteListedIPs == nil {
		apperrors.ClientError(ctx.Ctx, "no ip address whitelisted", nil, nil, ctx.DeviceID)
		return nil, false
	}
	if os.Getenv("ENV") == "production" {
		validIP := false
		for _, wIP := range *app.WhiteListedIPs {
			if wIP == ipAddress {
				validIP = true
				break
			}
		}
		if !validIP {
			apperrors.ClientError(ctx.Ctx, "unauthorised access", nil, nil, ctx.DeviceID)
			return nil, false
		}
	}

	ctx.SetContextData("AppID", appID)
	ctx.SetContextData("Name", app.Name)
	ctx.SetContextData("WorkspaceID", app.WorkspaceID)
	return ctx, true
}
