package middlewares

import (
	"fmt"
	"os"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/golang-jwt/jwt"
)

func OTPTokenMiddleware(ctx *interfaces.ApplicationContext[any], ipAddress string, intent string) (*interfaces.ApplicationContext[any], bool) {
	otpTokenPointer := ctx.GetHeader("X-Otp-Token")
	if otpTokenPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "missing otp token", ctx.DeviceID)
		return nil, false
	}
	otpToken := *otpTokenPointer
	validAccessToken, err := auth.DecodeAuthToken(otpToken)
	if err != nil {
		apperrors.AuthenticationError(ctx.Ctx, err.Error(), ctx.DeviceID)
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
	if authTokenClaims["iss"] != os.Getenv("JWT_ISSUER") {
		apperrors.AuthenticationError(ctx.Ctx, "this is not an authorized access token", ctx.DeviceID)
		return nil, false
	}
	var channel string
	if authTokenClaims["email"] != nil {
		channel = authTokenClaims["email"].(string)
	} else {
		channel = authTokenClaims["phoneNum"].(string)
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
	ctx.SetContextData("OTPPhone", authTokenClaims["phoneNum"])
	return ctx, true
}