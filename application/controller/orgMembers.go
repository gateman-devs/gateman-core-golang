package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	user_usecases "authone.usepolymer.co/application/usecases/user"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/logger"
	messagequeue "authone.usepolymer.co/infrastructure/message_queue"
	queue_tasks "authone.usepolymer.co/infrastructure/message_queue/tasks"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func InviteWorkspaceMembers(ctx *interfaces.ApplicationContext[dto.InviteWorspaceMembersDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	if len(ctx.Body.Invites) > 100 {
		apperrors.ClientError(ctx.Ctx, "You can only invite a maximum of 100 members at once", nil, nil)
		return
	}
	var wg sync.WaitGroup
	for _, invite := range ctx.Body.Invites {
		wg.Add(1)
		go func(invite dto.MemberInvite, workspaceID string, invitedBy string, workspaceName string) {
			defer wg.Done()

			payload, err := json.Marshal(queue_tasks.WorkspaceInvitePayload{
				Email:         invite.Email,
				WorkspaceName: workspaceName,
				WorkspaceID:   workspaceID,
				Permissions:   invite.Permissions,
				InvitedBy:     ctx.GetStringContextData("UserID"),
			})
			if err != nil {
				logger.Error("error marshalling payload for workspace invite queue")
				apperrors.FatalServerError(ctx.Ctx, err)
				return
			}
			messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
				Payload:   payload,
				Name:      queue_tasks.HandleWorkspaceInviteTaskName,
				Priority:  "high",
				ProcessIn: 1,
			})
		}(invite, ctx.GetStringContextData("WorkspaceID"), ctx.GetStringContextData("UserID"), ctx.GetStringContextData("WorkspaceName"))
	}
	wg.Wait()

	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "members invited", nil, nil, nil, nil, nil)
}

func ResendInvite(ctx *interfaces.ApplicationContext[dto.ResendWorspaceInviteDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	inviteRepo := repository.WorkspaceInviteRepo()
	invite, err := inviteRepo.FindByID(ctx.Body.ID)
	if err != nil {
		logger.Error("an error occured while trying to resend workspace invite", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.Body.ID,
		}, logger.LoggerOptions{
			Key:  "workspaceID",
			Data: ctx.GetStringContextData("WorkspaceID"),
		}, logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if invite == nil {
		apperrors.ClientError(ctx.Ctx, fmt.Sprintf("This email has not previously been invited to %s. Send a new invite to this email.", ctx.GetStringContextData("WorkspaceName")), nil, nil)
		return
	}
	if invite.Accepted != nil {
		decision := "accepted"
		if !*invite.Accepted {
			decision = "rejected"
		}
		apperrors.ClientError(ctx.Ctx, fmt.Sprintf("User has already %s the invite sent to them", decision), nil, nil)
		return
	}

	payload, err := json.Marshal(queue_tasks.EmailPayload{
		To:       invite.Email,
		Subject:  fmt.Sprintf("You have been invited to join %s", ctx.GetStringContextData("WorkspaceName")),
		Template: "workspace_invite",
	})
	if err != nil {
		logger.Error("error marshalling payload for email queue")
		apperrors.FatalServerError(ctx.Ctx, err)
		return
	}
	messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
		Payload:   payload,
		Name:      queue_tasks.HandleEmailDeliveryTaskName,
		Priority:  "high",
		ProcessIn: 1,
	})

	inviteRepo.UpdatePartialByID(ctx.Body.ID, map[string]any{
		"resentAt": time.Now(),
	})
}

func AcknowledgeWorkspaceInvite(ctx *interfaces.ApplicationContext[dto.AcknowledgeWorkspaceInviteDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	inviteRepo := repository.WorkspaceInviteRepo()
	invite, err := inviteRepo.FindByID(ctx.Body.ID)
	if err != nil {
		logger.Error("an error occured while trying to fetch invite for acknowledgement", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx.Body,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if invite == nil {
		apperrors.ClientError(ctx.Ctx, "invalid invite link", nil, nil)
		return
	}
	if invite.Accepted != nil {
		apperrors.ClientError(ctx.Ctx, "this link has already been used", nil, nil)
		return
	}
	workspaceMemberRepo := repository.WorkspaceMemberRepo()
	inviteRepo.UpdatePartialByID(ctx.Body.ID, map[string]any{
		"accepted": ctx.Body.Accepted,
	})
	if ctx.Body.Accepted {
		token, url, code, err := user_usecases.CreateUserUseCase(ctx.Ctx, &dto.CreateUserDTO{}, ctx.DeviceID, ctx.UserAgent, ctx.DeviceName)
		if err != nil {
			return
		}
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "authentication complete", map[string]any{
			"url":  url,
			"code": code,
		}, nil, nil, token, nil)

		workspaceMemberRepo.CreateOne(context.TODO(), entities.WorkspaceMember{
			WorkspaceID:   invite.WorkspaceID,
			WorkspaceName: invite.WorkspaceID,
			Permissions:   invite.Permissions,
		})
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "acknowledgement complete", nil, nil, nil, nil, nil)
}
