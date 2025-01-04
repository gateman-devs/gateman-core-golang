package middlewares

import (
	"fmt"
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
)

func UserAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], intent string, requiredPermissions *[]entities.MemberPermissions, workspaceSpecific bool) (*interfaces.ApplicationContext[any], bool) {
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
	validToken := cache.Cache.FindOne(fmt.Sprintf("%s-access", string(deviceIDHash)))
	if validToken == nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired")
		return nil, false
	}
	match := cryptography.CryptoHahser.VerifyHashData(*validToken, authToken)
	if !match {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired")
		return nil, false
	}

	if intent != "" {
		if authTokenClaims["intent"] != intent {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorised access")
			return nil, false
		}
	}
	if !authTokenClaims["verifiedAccount"].(bool) {
		apperrors.AuthenticationError(ctx.Ctx, "verify your account before trying to use this route")
		return nil, false
	}
	if authTokenClaims["tokenType"] != "access_token" {
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

	var workspaceName string
	var workspaceID string
	if workspaceSpecific {
		if ctx.GetHeader("X-Workspace-Id") == nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access")
			return nil, false
		} else {
			WorkspaceMemberRepo := repository.WorkspaceMemberRepo()
			orgMember, err := WorkspaceMemberRepo.FindOneByFilter(map[string]interface{}{
				"workspaceID": ctx.Header["X-Workspace-Id"][0],
				"userID":      authTokenClaims["userID"],
			})
			if err != nil {
				apperrors.AuthenticationError(ctx.Ctx, "unauthorized access")
				return nil, false
			}
			if orgMember == nil {
				apperrors.AuthenticationError(ctx.Ctx, "unauthorized access")
				return nil, false
			}
			if orgMember.Deactivated {
				apperrors.AuthenticationError(ctx.Ctx, "unauthorized access")
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
				apperrors.AuthenticationError(ctx.Ctx, "unauthorized access7")
				return nil, false
			}
			workspaceName = orgMember.WorkspaceName
			workspaceID = ctx.Header["X-Workspace-Id"][0]
		}
	}

	ctx.SetContextData("UserID", authTokenClaims["userID"])
	ctx.SetContextData("WorkspaceID", workspaceID)
	ctx.SetContextData("WorkspaceName", workspaceName)
	ctx.SetContextData("Email", authTokenClaims["email"])
	ctx.SetContextData("Phone", authTokenClaims["phone"])
	ctx.SetContextData("UserAgent", authTokenClaims["userAgent"])
	ctx.SetContextData("DeviceID", authTokenClaims["deviceID"])
	return ctx, true
}
