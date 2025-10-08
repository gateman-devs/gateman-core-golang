package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/constants"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	org_usecases "gateman.io/application/usecases/workspace"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/ipresolver"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
)

func CreateWorkspace(ctx *interfaces.ApplicationContext[dto.CreateWorkspaceDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	err := org_usecases.CreateWorkspaceUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.DeviceName, ctx.UserAgent, ctx.Param["ip"].(string))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, &ctx.DeviceID)
}

func UpdateOrgDetails(ctx *interfaces.ApplicationContext[dto.UpdateOrgDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	workspaceRepo := repository.WorkspaceRepository()
	workspaceRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": ctx.GetStringContextData("WorkspaceID"),
	}, ctx.Body)
	if ctx.Body.WorkspaceName != nil {
		workspaceMemberRepo := repository.WorkspaceMemberRepo()
		workspaceMemberRepo.UpdatePartialByFilter(map[string]interface{}{
			"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		}, map[string]any{
			"workspaceName": ctx.Body.WorkspaceName,
		})
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org updated", nil, nil, nil, &ctx.DeviceID)
}

func FetchWorkspaceDetails(ctx *interfaces.ApplicationContext[any]) {
	workspaceRepo := repository.WorkspaceRepository()
	workspace, _ := workspaceRepo.FindOneByFilter(map[string]interface{}{
		"_id": ctx.GetStringContextData("WorkspaceID"),
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "workspace details fetched", workspace, nil, nil, &ctx.DeviceID)
}

func LoginWorkspaceMember(ctx *interfaces.ApplicationContext[dto.LoginWorkspaceMemberDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	workspaceMemberRepo := repository.WorkspaceMemberRepo()
	member, err := workspaceMemberRepo.FindOneByFilter(map[string]interface{}{
		"email": ctx.Body.Email,
	})
	if err != nil {
		logger.Error("an error occured while trying to fetch a workspace member for login", logger.LoggerOptions{
			Key:  "email",
			Data: ctx.Body.Email,
		}, logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if member == nil {
		apperrors.NotFoundError(ctx.Ctx, "incorrect email or password", &ctx.DeviceID)
		return
	}
	passwordResult := cryptography.CryptoHahser.VerifyHashData(member.Password, ctx.Body.Password)
	if !passwordResult {
		apperrors.AuthenticationError(ctx.Ctx, "incorrect email or password", ctx.DeviceID)
		return
	}
	if !member.VerifiedEmail {
		otp, err := auth.GenerateOTP(6, ctx.Body.Email)
		if err != nil {
			apperrors.FatalServerError(ctx, err, ctx.DeviceID)
			return
		}
		emailPayload, err := json.Marshal(queue_tasks.EmailPayload{
			Opts: map[string]any{
				"OTP": otp,
			},
			To:       ctx.Body.Email,
			Subject:  "Gateman OTP",
			Template: "workspace_created",
			Intent:   "verify_workspace",
		})
		if err != nil {
			logger.Error("error marshalling payload for email queue")
			apperrors.FatalServerError(ctx, err, ctx.DeviceID)
			return
		}
		messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
			Payload:   emailPayload,
			Name:      queue_tasks.HandleEmailDeliveryTaskName,
			Priority:  mq_types.High,
			ProcessIn: 1,
		})
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", ctx.Body.Email), "verify_workspace", time.Minute*10)
		apperrors.ClientError(ctx.Ctx, "Veify your email", nil, &constants.VERIFY_WORKSPACE_MEMBER_EMAIL, ctx.DeviceID)
		return
	}

	var savedDevice *entities.Device
	for i, device := range member.Devices {
		if device.ID == ctx.DeviceID {
			savedDevice = &member.Devices[i]
			member.Devices = append(member.Devices[:i], member.Devices[i+1:]...)
			break
		}
	}
	if savedDevice == nil {
		ipLookupRes, _ := ipresolver.IPResolverInstance.LookUp(ctx.Param["ip"].(string))
		member.Devices = append(member.Devices, entities.Device{
			ID:                ctx.DeviceID,
			Name:              ctx.DeviceName,
			LastLogin:         time.Now(),
			LastLoginLocation: fmt.Sprintf("%s, %s - (%f, %f)", strings.ToUpper(ipLookupRes.City), strings.ToUpper(ipLookupRes.CountryCode), ipLookupRes.Longitude, ipLookupRes.Latitude),
			Verified:          true,
		})
	} else {
		ipLookupRes, _ := ipresolver.IPResolverInstance.LookUp(ctx.Param["ip"].(string))
		member.Devices = append(member.Devices, entities.Device{
			ID:                savedDevice.ID,
			Name:              savedDevice.Name,
			LastLogin:         time.Now(),
			LastLoginLocation: fmt.Sprintf("%s, %s - (%f, %f)", strings.ToUpper(ipLookupRes.City), strings.ToUpper(ipLookupRes.CountryCode), ipLookupRes.Longitude, ipLookupRes.Latitude),
			Verified:          true,
		})
	}
	workspaceMemberRepo.UpdatePartialByFilter(map[string]interface{}{
		"email": ctx.Body.Email,
	}, map[string]any{
		"devices": member.Devices,
	})

	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          member.ID,
		UserAgent:       member.UserAgent,
		Email:           &member.Email,
		VerifiedAccount: member.VerifiedEmail,
		WorkspaceID:     &member.WorkspaceID,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	refreshToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          member.ID,
		UserAgent:       member.UserAgent,
		Email:           &member.Email,
		VerifiedAccount: member.VerifiedEmail,
		TokenType:       auth.RefreshToken,
		WorkspaceID:     &member.WorkspaceID,
		DeviceID:        ctx.DeviceID,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 180).Unix(), //lasts for 180 days
	})

	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-workspace-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)       // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-workspace-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 100 days
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "login successful", map[string]any{
		"workspaceAccessToken":  accessToken,
		"workspaceRefreshToken": refreshToken,
		"profile":               member,
	}, nil, nil, &ctx.DeviceID)
}

func ResetWorkspaceMemberPassword(ctx *interfaces.ApplicationContext[dto.ResetWorkspaceMemberPasswordDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}

	if ctx.Body.Email != nil {
		workspaceMemberRepo := repository.WorkspaceMemberRepo()
		member, err := workspaceMemberRepo.FindOneByFilter(map[string]interface{}{
			"email": ctx.Body.Email,
		})
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		if member == nil {
			apperrors.NotFoundError(ctx.Ctx, "member not found", &ctx.DeviceID)
			return
		}

		passwordResult := cryptography.CryptoHahser.VerifyHashData(member.Password, ctx.Body.CurrentPassword)
		if !passwordResult {
			apperrors.AuthenticationError(ctx.Ctx, "incorrect password", ctx.DeviceID)
			return
		}
		password, err := cryptography.CryptoHahser.HashString(ctx.Body.NewPassword, nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		workspaceMemberRepo.UpdatePartialByID(member.ID, map[string]any{
			"password": string(password),
		})
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "password reset successfully", nil, nil, nil, &ctx.DeviceID)
		return
	}
	workspaceMemberRepo := repository.WorkspaceMemberRepo()
	member, err := workspaceMemberRepo.FindOneByFilter(map[string]interface{}{
		"email": ctx.Body.Email,
	})
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if member == nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "password reset email sent", nil, nil, nil, &ctx.DeviceID)
		return
	}
	otp, err := auth.GenerateOTP(6, *ctx.Body.Email)
	if err != nil {
		apperrors.FatalServerError(ctx, err, ctx.DeviceID)
		return
	}
	cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *ctx.Body.Email), "reset_workspace_member_password", time.Minute*10)

	emailPayload, err := json.Marshal(queue_tasks.EmailPayload{
		Opts: map[string]any{
			"OTP_CODE":       otp,
			"RECIPIENT_NAME": member.LastName,
			"REQUEST_ACTION": "reset password",
			"APP_NAME":       member.WorkspaceName,
			"EXPIRY_MINUTES": "10",
		},
		To:       *ctx.Body.Email,
		Subject:  "Gateman OTP",
		Template: "otp-request",
		Intent:   "reset_workspace_member_password",
	})
	if err != nil {
		logger.Error("error marshalling payload for email queue")
		apperrors.FatalServerError(ctx, err, ctx.DeviceID)
		return
	}
	messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
		Payload:   emailPayload,
		Name:      queue_tasks.HandleEmailDeliveryTaskName,
		Priority:  mq_types.High,
		ProcessIn: 1,
	})
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "password reset email sent", nil, nil, nil, &ctx.DeviceID)
}
