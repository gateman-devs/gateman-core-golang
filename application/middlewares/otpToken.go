package middlewares

import (
	"fmt"
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"github.com/golang-jwt/jwt"
)

func OTPTokenMiddleware(ctx *interfaces.ApplicationContext[any], ipAddress string, intent string, otpToken string) (*interfaces.ApplicationContext[any], bool) {
	if otpToken == "" {
		apperrors.AuthenticationError(ctx.Ctx, "missing otp token", ctx.DeviceID)
		return nil, false
	}
	validAccessToken, err := auth.DecodeAuthToken(otpToken)
	if err != nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	if !validAccessToken.Valid {
		apperrors.AuthenticationError(ctx.Ctx, "invalid access token used", ctx.DeviceID)
		return nil, false
	}
	invalidToken := cache.Cache.FindOne(otpToken)
	if invalidToken != nil {
		apperrors.AuthenticationError(ctx.Ctx, "expired access token used", ctx.DeviceID)
		return nil, false
	}
	authTokenClaims := validAccessToken.Claims.(jwt.MapClaims)
	if authTokenClaims["iss"] != os.Getenv("GATEMAN_ISSUER") {
		apperrors.AuthenticationError(ctx.Ctx, "this is not an authorized access token", ctx.DeviceID)
		return nil, false
	}
	var channel string
	if authTokenClaims["email"] != nil {
		channel = authTokenClaims["email"].(string)
	} else {
		channel = authTokenClaims["phone"].(string)
	}
	otpIntent := cache.Cache.FindOne(fmt.Sprintf("%s-otp-intent", channel))
	if otpIntent == nil {
		logger.Error("otp intent missing")
		apperrors.ClientError(ctx.Ctx, "otp expired", nil, nil, ctx.DeviceID)
		return nil, false
	}

	if *otpIntent != authTokenClaims["intent"].(string) || authTokenClaims["intent"].(string) != intent {
		logger.Error("wrong otp intent in token")
		apperrors.ClientError(ctx.Ctx, "incorrect intent", nil, nil, ctx.DeviceID)
		return nil, false
	}
	ctx.SetContextData("OTPToken", otpToken)
	ctx.SetContextData("OTPEmail", authTokenClaims["email"])
	ctx.SetContextData("OTPPhone", authTokenClaims["phone"])
	return ctx, true
}
