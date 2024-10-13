package controller

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	org_usecases "authone.usepolymer.co/application/usecases/organisation"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/messaging/emails"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func CreateOrganisation(ctx *interfaces.ApplicationContext[dto.CreateOrgDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	err := org_usecases.CreateOrgUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("Email"))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, ctx.DeviceID)
}

func FetchWorkspaces(ctx *interfaces.ApplicationContext[dto.CreateOrgDTO]) {
	WorkspaceMemberRepo := repository.WorkspaceMemberRepo()
	workspaces, err := WorkspaceMemberRepo.FindMany(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
	})
	if err != nil {
		logger.Error("error fetching users orgs", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "workspaces fetched", workspaces, nil, nil, ctx.DeviceID)
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
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org updated", nil, nil, nil, ctx.DeviceID)
}

func InviteWorkspaceMembers(ctx *interfaces.ApplicationContext[dto.InviteWorspaceMembersDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if len(ctx.Body.Emails) > 100 {
		apperrors.ClientError(ctx.Ctx, "You can only invite a maximum of 100 members at once", nil, nil, ctx.DeviceID)
		return
	}
	var wg sync.WaitGroup
	inviteRepo := repository.WorkspaceInviteRepo()
	invitePayloadChan := make(chan entities.WorkspaceInvite)
	for _, email := range ctx.Body.Emails {
		wg.Add(1)
		go func(email string, workspaceID string, invitedBy string) {
			defer wg.Done()

			inviteRepo := repository.WorkspaceInviteRepo()
			inviteExists, err := inviteRepo.CountDocs(map[string]interface{}{
				"workspaceID": workspaceID,
				"email":       email,
			})
			if err != nil {
				logger.Error("an error occured while trying to check if invite exists", logger.LoggerOptions{
					Key:  "err",
					Data: err.Error(),
				}, logger.LoggerOptions{
					Key:  "email",
					Data: email,
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
				Email:       email,
				WorkspaceID: workspaceID,
				InvitedByID: invitedBy,
			}
			emails.EmailService.SendEmail(email, fmt.Sprintf("You have been invited to join %s", ctx.GetStringContextData("WorkspaceName")), "workspace_invite", nil)
		}(email, ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("UserID"))
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
	
}
