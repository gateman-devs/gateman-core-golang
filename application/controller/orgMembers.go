package controller

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	auth_usecases "authone.usepolymer.co/application/usecases/auth"
	user_usecases "authone.usepolymer.co/application/usecases/user"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/messaging/emails"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func InviteWorkspaceMembers(ctx *interfaces.ApplicationContext[dto.InviteWorspaceMembersDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if len(ctx.Body.Invites) > 100 {
		apperrors.ClientError(ctx.Ctx, "You can only invite a maximum of 100 members at once", nil, nil, ctx.DeviceID)
		return
	}
	var wg sync.WaitGroup
	inviteRepo := repository.WorkspaceInviteRepo()
	invitePayloadChan := make(chan entities.WorkspaceInvite)
	for _, invite := range ctx.Body.Invites {
		wg.Add(1)
		go func(invite dto.MemberInvite, workspaceID string, invitedBy string, workspaceName string) {
			defer wg.Done()

			inviteRepo := repository.WorkspaceInviteRepo()
			inviteExists, err := inviteRepo.CountDocs(map[string]interface{}{
				"workspaceID": workspaceID,
				"email":       invite.Email,
			})
			if err != nil {
				logger.Error("an error occured while trying to check if invite exists", logger.LoggerOptions{
					Key:  "err",
					Data: err.Error(),
				}, logger.LoggerOptions{
					Key:  "invite",
					Data: invite,
				}, logger.LoggerOptions{
					Key: "workspaceID", Data: workspaceID,
				})
				return
			}
			if inviteExists != 0 {
				// resend email
				return
			}
			invitePayloadChan <- entities.WorkspaceInvite{
				Email:         invite.Email,
				WorkspaceID:   workspaceID,
				Permissions:   invite.Permissions,
				InvitedByID:   invitedBy,
				WorkspaceName: workspaceName,
			}
			emails.EmailService.SendEmail(invite.Email, fmt.Sprintf("You have been invited to join %s", ctx.GetStringContextData("WorkspaceName")), "workspace_invite", nil)
		}(invite, ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceName"))
	}
	wg.Wait()
	invitePayloadArray := []entities.WorkspaceInvite{}
	for invite := range invitePayloadChan {
		invitePayloadArray = append(invitePayloadArray, invite)
	}
	_, err := inviteRepo.CreateBulk(invitePayloadArray)
	if err != nil {
		logger.Error("an error occured while trying to create invites", logger.LoggerOptions{
			Key:  "err",
			Data: err.Error(),
		}, logger.LoggerOptions{
			Key:  "user",
			Data: ctx.GetStringContextData("UserID"),
		})
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "members invited", nil, nil, nil, ctx.DeviceID)
}

func ResendInvite(ctx *interfaces.ApplicationContext[dto.ResendWorspaceInviteDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	inviteRepo := repository.WorkspaceInviteRepo()
	invite, err := inviteRepo.FindOneByFilter(map[string]interface{}{
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		"email":       ctx.Body.Email,
	})
	if err != nil {
		logger.Error("an error occured while trying to resend workspace invite", logger.LoggerOptions{
			Key:  "email",
			Data: ctx.Body.Email,
		}, logger.LoggerOptions{
			Key:  "workspaceID",
			Data: ctx.GetStringContextData("WorkspaceID"),
		}, logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if invite == nil {
		apperrors.ClientError(ctx.Ctx, fmt.Sprintf("This email has not previously been invited to %s. Send a new invite to this email.", ctx.GetStringContextData("WorkspaceName")), nil, nil, ctx.DeviceID)
		return
	}
	if invite.Accepted != nil {
		apperrors.ClientError(ctx.Ctx, "User has already rejected the incinte sent to them", nil, nil, ctx.DeviceID)
		return
	}
	emails.EmailService.SendEmail(invite.Email, fmt.Sprintf("You have been invited to join %s", ctx.GetStringContextData("WorkspaceName")), "workspace_invite", nil)
	inviteRepo.UpdatePartialByFilter(map[string]interface{}{
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		"email":       ctx.Body.Email,
	}, map[string]any{
		"resentAt": time.Now(),
	})
}

func AcknowledgeWorkspaceInvite(ctx *interfaces.ApplicationContext[dto.AcknowledgeWorkspaceInviteDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	inviteRepo := repository.WorkspaceInviteRepo()
	invite, err := inviteRepo.FindByID(ctx.Body.ID)
	if err != nil {
		logger.Error("an error occured while trying to fetch invite for acknowledgement", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if invite == nil {
		apperrors.ClientError(ctx.Ctx, "invalid invite link", nil, nil, ctx.DeviceID)
		return
	}
	if invite.Accepted != nil {
		apperrors.ClientError(ctx.Ctx, "this link has already been used", nil, nil, ctx.DeviceID)
		return
	}
	workspaceMemberRepo := repository.WorkspaceMemberRepo()
	workspaceCount, err := workspaceMemberRepo.CountDocs(map[string]any{
		"email": invite.Email,
	})
	if err != nil {
		logger.Error("something went wrong while trying to count user workspaces to acknowledge invite", logger.LoggerOptions{
			Key:  "invite",
			Data: invite,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if workspaceCount >= 20 {
		apperrors.ClientError(ctx.Ctx, "you have reached the maximum of number of workspaces", nil, nil, ctx.DeviceID)
		return
	}
	inviteRepo.UpdatePartialByID(ctx.Body.ID, map[string]any{
		"accepted": ctx.Body.Accepted,
	})
	if ctx.Body.Accepted {
		serverPublicKey, encryptedSecret := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.Body.ClientPublicKey, ctx.DeviceID)
		token, url, code, err := user_usecases.CreateUserUseCase(ctx.Ctx, &dto.CreateUserDTO{}, ctx.DeviceID, ctx.UserAgent, encryptedSecret, ctx.DeviceName)
		if err != nil {
			return
		}
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "authentication complete", map[string]any{
			"serverPublicKey": hex.EncodeToString(serverPublicKey),
			"url":             url,
			"code":            code,
			"token":           token,
		}, nil, nil, ctx.DeviceID)

		workspaceMemberRepo.CreateOne(context.TODO(), entities.WorkspaceMember{
			WorkspaceID:   invite.WorkspaceID,
			WorkspaceName: invite.WorkspaceID,
			Permissions:   invite.Permissions,
		})
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "acknowledgement complete", nil, nil, nil, ctx.DeviceID)
}
