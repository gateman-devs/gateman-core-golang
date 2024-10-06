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

func UserAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], intent string, requiredPermissions *[]entities.MemberPermissions) (*interfaces.ApplicationContext[any], bool) {
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
	authTokenClaims := valid_access_token.Claims.(jwt.MapClaims)
	if authTokenClaims["iss"] != os.Getenv("JWT_ISSUER") {
		logger.Warning("attempt to access account with tampered jwt", logger.LoggerOptions{
			Key:  "token claims",
			Data: valid_access_token,
		})
		apperrors.AuthenticationError(ctx.Ctx, "invalid access token used", ctx.DeviceID)
		return nil, false
	}

	valid_token := cache.Cache.FindOne(authTokenClaims["userID"].(string))
	if valid_token == nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	match := cryptography.CryptoHahser.VerifyHashData(*valid_token, auth_token)
	if !match {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}

	if intent != "" {
		if authTokenClaims["intent"] != intent {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorised access", ctx.DeviceID)
			return nil, false
		}
	}
	userRepo := repository.UserRepo()
	account, err := userRepo.FindByID(authTokenClaims["userID"].(string), options.FindOne().SetProjection(map[string]any{
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

	if intent != "face_verification" && !account.VerifiedAccount {
		apperrors.AuthenticationError(ctx.Ctx, "verify your account before trying to use this route", ctx.DeviceID)
		return nil, false
	}

	if ctx.DeviceID == nil {
		auth.SignOutUser(ctx.Ctx, account.ID, "client made request without a device id")
		logger.Info("device id missing from client")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
		return nil, false
	}

	if authTokenClaims["deviceID"] != *ctx.DeviceID {
		logger.Warning("client made request using device id different from that in access token", logger.LoggerOptions{
			Key:  "token device id",
			Data: authTokenClaims["deviceID"],
		}, logger.LoggerOptions{
			Key:  "request  device id",
			Data: ctx.DeviceID,
		})
		auth.SignOutUser(ctx.Ctx, account.ID, "client made request using device id different from that in access token")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
		return nil, false
	}

	if ctx.Param["orgID"] != "" {
		orgMemberRepo := repository.OrgMemberRepo()
		orgID := ctx.Param["orgID"]
		orgMember, err := orgMemberRepo.FindOneByFilter(map[string]interface{}{
			"orgID":  orgID,
			"userID": authTokenClaims["userID"],
		})
		if err != nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		if orgMember == nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		if orgMember.Deactivated {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		hasAccess := false
		for _, rPermission := range *requiredPermissions {
			hasPermission := false
			for _, uPermission := range orgMember.Permissions {
				if uPermission == entities.SUPER_ACCESS || uPermission == rPermission {
					hasPermission = true
					break
				}
			}
			if hasPermission {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
	}

	ctx.SetContextData("UserID", authTokenClaims["userID"])
	ctx.SetContextData("OrgID", authTokenClaims["orgID"])
	ctx.SetContextData("Email", authTokenClaims["email"])
	ctx.SetContextData("Phone", authTokenClaims["phone"])
	ctx.SetContextData("UserAgent", authTokenClaims["userAgent"])
	ctx.SetContextData("DeviceID", authTokenClaims["deviceID"])
	return ctx, true
}
