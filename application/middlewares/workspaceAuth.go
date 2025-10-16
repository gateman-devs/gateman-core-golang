package middlewares

import (
	"fmt"
	"os"

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

func WorkspaceAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], intent *string, requiredPermissions *[]entities.MemberPermissions, authToken string) (*interfaces.ApplicationContext[any], bool) {
	validAccessToken, err := auth.DecodeAuthToken(authToken)
	fmt.Println("the token")
	fmt.Println(authToken)
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
	validToken := cache.Cache.FindOne(fmt.Sprintf("%s-workspace-access", string(deviceIDHash)))
	if validToken == nil {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}
	match := cryptography.CryptoHahser.VerifyHashData(*validToken, authToken)
	if !match {
		apperrors.AuthenticationError(ctx.Ctx, "this session has expired", ctx.DeviceID)
		return nil, false
	}

	if intent != nil {
		if authTokenClaims["intent"] != intent {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorised access", ctx.DeviceID)
			return nil, false
		}
	}
	if !authTokenClaims["verifiedAccount"].(bool) {
		apperrors.AuthenticationError(ctx.Ctx, "verify your account before trying to use this route", ctx.DeviceID)
		return nil, false
	}
	if authTokenClaims["tokenType"] != "access_token" {
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

	var workspaceName string
	var workspaceEmail string
	if authTokenClaims["workspace"] == nil {
		apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
		return nil, false
	} else {
		WorkspaceMemberRepo := repository.WorkspaceMemberRepo()
		workspaceMember, err := WorkspaceMemberRepo.FindOneByFilter(map[string]interface{}{
			"workspaceID": authTokenClaims["workspace"],
		})
		if err != nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		if workspaceMember == nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		if workspaceMember.Deactivated {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
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
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		workspaceRepo := repository.WorkspaceRepository()
		workspace, err := workspaceRepo.FindByID(authTokenClaims["workspace"].(string), options.FindOne().SetProjection(map[string]any{
			"email": 1,
		}))
		if err != nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
			return nil, false
		}
		if workspace == nil {
			apperrors.AuthenticationError(ctx.Ctx, "unauthorized access", ctx.DeviceID)
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
	ctx.SetContextData("DeviceID", ctx.DeviceID)
	return ctx, true
}
