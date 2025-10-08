package middlewares

import (
	"fmt"
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"github.com/golang-jwt/jwt"
)

func RefreshTokenMiddleware(ctx *interfaces.ApplicationContext[any], workspaceToken bool, authToken string) (*interfaces.ApplicationContext[any], bool) {
	if authToken == "" {
		apperrors.AuthenticationError(ctx.Ctx, "missing auth token", ctx.DeviceID)
		return nil, false
	}
	validAccessToken, err := auth.DecodeAuthToken(authToken)
	if err != nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	if !validAccessToken.Valid {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access", ctx.DeviceID)
		return nil, false
	}
	authTokenClaims := validAccessToken.Claims.(jwt.MapClaims)
	if authTokenClaims["iss"] != os.Getenv("GATEMAN_ISSUER") {
		logger.Warning("attempt to access account with tampered jwt", logger.LoggerOptions{
			Key:  "token claims",
			Data: validAccessToken,
		})
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access", ctx.DeviceID)
		return nil, false
	}

	deviceIDHash, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	var tokenKey string
	if workspaceToken {
		tokenKey = fmt.Sprintf("%s-workspace-refresh", string(deviceIDHash))
	} else {
		tokenKey = fmt.Sprintf("%s-refresh", string(deviceIDHash))
	}
	validToken := cache.Cache.FindOne(tokenKey)
	if validToken == nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	match := cryptography.CryptoHahser.VerifyHashData(*validToken, authToken)
	if !match {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}

	if !authTokenClaims["verifiedAccount"].(bool) {
		apperrors.AuthenticationError(ctx.Ctx, "verify your account before trying to use this route", ctx.DeviceID)
		return nil, false
	}
	if authTokenClaims["tokenType"] != "refresh_token" {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access", ctx.DeviceID)
		return nil, false
	}

	if ctx.DeviceID == "" {
		logger.Info("device id missing from client")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
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
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
		return nil, false
	}

	ctx.SetContextData("UserID", authTokenClaims["userID"])
	ctx.SetContextData("Email", authTokenClaims["email"])
	ctx.SetContextData("Phone", authTokenClaims["phone"])
	return ctx, true
}
