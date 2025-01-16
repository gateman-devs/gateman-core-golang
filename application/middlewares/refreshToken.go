package middlewares

import (
	"fmt"
	"os"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/golang-jwt/jwt"
)

func RefreshTokenMiddleware(ctx *interfaces.ApplicationContext[any]) (*interfaces.ApplicationContext[any], bool) {
	authTokenHeaderPointer := ctx.GetHeader("Authorization")
	if authTokenHeaderPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an auth token")
		return nil, false
	}
	authTokenHeader := *authTokenHeaderPointer
	authToken := strings.Split(authTokenHeader, " ")[1]
	validAccessToken, err := auth.DecodeAuthToken(authToken)
	if err != nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired")
		return nil, false
	}
	if !validAccessToken.Valid {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access")
		return nil, false
	}
	authTokenClaims := validAccessToken.Claims.(jwt.MapClaims)
	if authTokenClaims["iss"] != os.Getenv("JWT_ISSUER") {
		logger.Warning("attempt to access account with tampered jwt", logger.LoggerOptions{
			Key:  "token claims",
			Data: validAccessToken,
		})
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access")
		return nil, false
	}

	deviceIDHash, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	validToken := cache.Cache.FindOne(fmt.Sprintf("%s-refresh", string(deviceIDHash)))
	if validToken == nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired")
		return nil, false
	}
	match := cryptography.CryptoHahser.VerifyHashData(*validToken, authToken)
	if !match {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired")
		return nil, false
	}

	if !authTokenClaims["verifiedAccount"].(bool) {
		apperrors.AuthenticationError(ctx.Ctx, "verify your account before trying to use this route")
		return nil, false
	}
	if authTokenClaims["tokenType"] != "refresh_token" {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access")
		return nil, false
	}

	if ctx.DeviceID == "" {
		logger.Info("device id missing from client")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access")
		return nil, false
	}

	if authTokenClaims["deviceID"] != ctx.DeviceID {
		logger.Warning("client made request using device id different from that in access token", logger.LoggerOptions{
			Key:  "token device id",
			Data: authTokenClaims["deviceID"],
		}, logger.LoggerOptions{
			Key:  "request  device id",
			Data: ctx.DeviceID,
		})
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access")
		return nil, false
	}

	ctx.SetContextData("UserID", authTokenClaims["userID"])
	ctx.SetContextData("Email", authTokenClaims["email"])
	ctx.SetContextData("Phone", authTokenClaims["phone"])
	return ctx, true
}
