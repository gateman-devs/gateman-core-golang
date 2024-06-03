package middlewares

import (
	"os"
	"strings"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// set verifiedOnly to true to allow only users who have verified their emails
func AuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], verifiedOnly bool, requiredPermisison *[]entities.MemberPermissions) (*interfaces.ApplicationContext[any], bool) {
	authTokenHeaderPointer := ctx.GetHeader("Authorization")
	if authTokenHeaderPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an auth token", ctx.DeviceID)
		return nil, false
	}
	authTokenHeader := *authTokenHeaderPointer
	auth_token := strings.Split(authTokenHeader, " ")[1]
	valid_access_token, err := auth.DecodeAuthToken(auth_token)
	if err != nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	if !valid_access_token.Valid {
		apperrors.AuthenticationError(ctx.Ctx, "invalid access token used", ctx.DeviceID)
		return nil, false
	}
	auth_token_claims := valid_access_token.Claims.(jwt.MapClaims)
	if auth_token_claims["iss"] != os.Getenv("JWT_ISSUER") {
		logger.Warning("attempt to access account with tampered jwt", logger.LoggerOptions{
			Key:  "token claims",
			Data: valid_access_token,
		})
		apperrors.AuthenticationError(ctx.Ctx, "invalid access token used", ctx.DeviceID)
		return nil, false
	}

	valid_token := cache.Cache.FindOne(auth_token_claims["userID"].(string))
	if valid_token == nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	match := cryptography.CryptoHahser.VerifyHashData(*valid_token, auth_token)
	if !match {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	orgMemberRepo := repository.OrgMemberRepo()
	account, err := orgMemberRepo.FindByID(auth_token_claims["userID"].(string), options.FindOne().SetProjection(map[string]any{
		"permissions":   1,
		"verifiedEmail": 1,
		"deactivated":   1,
		"deviceID":      1,
	}))
	if err != nil {
		apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
		return nil, false
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "this account no longer exists", ctx.DeviceID)
		return nil, false
	}
	if account.Deactivated {
		apperrors.AuthenticationError(ctx.Ctx, "your account has been deactivated. contact your administrator if this is a mistake", ctx.DeviceID)
		return nil, false
	}

	if !account.VerifiedEmail {
		apperrors.AuthenticationError(ctx.Ctx, "verify your email before trying to log in", ctx.DeviceID)
		return nil, false
	}

	if ctx.DeviceID == nil {
		auth.SignOutUser(ctx.Ctx, account.ID, "client made request without a device id")
		logger.Info("device id missing from client")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
		return nil, false
	}
	if auth_token_claims["deviceID"] != account.DeviceID || account.DeviceID != *ctx.DeviceID || auth_token_claims["deviceID"] != *ctx.DeviceID {
		logger.Warning("client made request using device id different from that in access token", logger.LoggerOptions{
			Key:  "token device id",
			Data: auth_token_claims["deviceID"],
		}, logger.LoggerOptions{
			Key:  "client  device id",
			Data: account.DeviceID,
		}, logger.LoggerOptions{
			Key:  "request  device id",
			Data: ctx.DeviceID,
		})
		logger.Warning("this triggers a wallet lock")
		auth.SignOutUser(ctx.Ctx, account.ID, "client made request using device id different from that in access token")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
		return nil, false
	}

	ctx.SetContextData("UserID", auth_token_claims["userID"])
	ctx.SetContextData("OrgID", auth_token_claims["orgID"])
	ctx.SetContextData("LastName", account.LastName)
	ctx.SetContextData("FirstName", account.FirstName)
	ctx.SetContextData("Email", auth_token_claims["email"])
	ctx.SetContextData("UserAgent", auth_token_claims["userAgent"])
	ctx.SetContextData("DeviceID", auth_token_claims["deviceID"])
	return ctx, true
}
