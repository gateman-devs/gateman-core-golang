package middlewares

import (
	"os"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/infrastructure/cryptography"
)

func AppAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], ipAddress string) (*interfaces.ApplicationContext[any], bool) {
	apiKeyPointer := ctx.GetHeader("X-Api-Key")
	if apiKeyPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an api key")
		return nil, false
	}
	apiKey := *apiKeyPointer
	appIDPointer := ctx.GetHeader("X-App-Id")
	if appIDPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an app id")
		return nil, false
	}
	appID := *appIDPointer
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": appID,
	})
	if app == nil {
		apperrors.NotFoundError(ctx.Ctx, "invalid credentials")
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
		apperrors.ClientError(ctx.Ctx, "invalid credentials", nil, nil)
		return nil, false
	}
	if app.WhiteListedIPs == nil {
		apperrors.ClientError(ctx.Ctx, "no ip address whitelisted", nil, nil)
		return nil, false
	}
	validIP := false
	if os.Getenv("ENV") != "production" {
		validIP = true
	} else {
		for _, wIP := range *app.WhiteListedIPs {
			if wIP == ipAddress {
				validIP = true
				break
			}
		}
		if !validIP {
			apperrors.ClientError(ctx.Ctx, "unauthorised access", nil, nil)
			return nil, false
		}
	}

	ctx.SetContextData("AppID", app.ID)
	ctx.SetContextData("Name", app.Name)
	ctx.SetContextData("WorkspaceID", app.WorkspaceID)
	return ctx, true
}
