package middlewares

import (
	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	authusecase "gateman.io/application/usecases/auth"
)

func UserAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], intent *string, authToken string) (*interfaces.ApplicationContext[any], bool) {
	authResult := authusecase.IsUserSignedIn(ctx.Ctx, authToken, intent, *ctx.GetHeader("X-Device-Id"))

	if !authResult.IsAuthenticated {
		apperrors.AuthenticationError(ctx.Ctx, authResult.ErrorMessage, *ctx.GetHeader("X-Device-Id"))
		return nil, false
	}

	// Set user context data from the authentication result
	ctx.SetContextData("UserID", authResult.UserID)
	ctx.SetContextData("Email", authResult.Email)
	ctx.SetContextData("Phone", authResult.Phone)
	ctx.SetContextData("UserAgent", authResult.UserAgent)
	ctx.SetContextData("DeviceID", authResult.DeviceID)

	return ctx, true
}
