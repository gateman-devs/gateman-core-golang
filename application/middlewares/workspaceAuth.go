package middlewares

import (
	"fmt"
	"os"
	"strings"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"github.com/golang-jwt/jwt"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func WorkspaceAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], intent *string, requiredPermissions *[]entities.MemberPermissions) (*interfaces.ApplicationContext[any], bool) {
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
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access1")
		return nil, false
	}
	authTokenClaims := validAccessToken.Claims.(jwt.MapClaims)
	if authTokenClaims["iss"] != os.Getenv("GATEMAN_ISSUER") {
		logger.Warning("attempt to access account with tampered jwt", logger.LoggerOptions{
			Key:  "token claims",
			Data: validAccessToken,
		})
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access2")
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

	if intent != nil {
		if authTokenClaims["intent"] != intent {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorised access3")
			return nil, false
		}
	}
	if !authTokenClaims["verifiedAccount"].(bool) {
		apperrors.AuthenticationError(ctx.Ctx, "verify your account before trying to use this route")
		return nil, false
	}
	if authTokenClaims["tokenType"] != "access_token" {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorised access4")
		return nil, false
	}

	if ctx.DeviceID == "" {
		logger.Info("device id missing from client")
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access1")
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
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access2")
		return nil, false
	}

	var workspaceName string
	var workspaceEmail string
	if authTokenClaims["workspace"] == nil {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access3")
		return nil, false
	} else {
		WorkspaceMemberRepo := repository.WorkspaceMemberRepo()
		workspaceMember, err := WorkspaceMemberRepo.FindOneByFilter(map[string]interface{}{
			"workspaceID": authTokenClaims["workspace"],
			"userID":      authTokenClaims["userID"],
		})
		if err != nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access4")
			return nil, false
		}
		if workspaceMember == nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access5")
			return nil, false
		}
		if workspaceMember.Deactivated {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access6")
			return nil, false
		}
		hasAccess := false
		for _, rPermission := range *requiredPermissions {
			hasPermission := false
			for _, uPermission := range workspaceMember.Permissions {
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
		workspaceRepo := repository.WorkspaceRepository()
		workspace, err := workspaceRepo.FindByID(authTokenClaims["workspace"].(string), options.FindOne().SetProjection(map[string]any{
			"email": 1,
		}))
		if err != nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access8")
			return nil, false
		}
		if workspace == nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access9")
			return nil, false
		}
		workspaceName = workspaceMember.WorkspaceName
		workspaceEmail = workspace.Email
	}

	ctx.SetContextData("UserID", authTokenClaims["userID"])
	ctx.SetContextData("WorkspaceID", authTokenClaims["workspace"])
	ctx.SetContextData("WorkspaceName", workspaceName)
	ctx.SetContextData("WorkspaceEmail", workspaceEmail)
	ctx.SetContextData("Email", authTokenClaims["email"])
	ctx.SetContextData("UserAgent", authTokenClaims["userAgent"])
	ctx.SetContextData("DeviceID", authTokenClaims["deviceID"])
	return ctx, true
}
