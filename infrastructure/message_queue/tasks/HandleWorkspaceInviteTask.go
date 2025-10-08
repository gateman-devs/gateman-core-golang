package queue_tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gateman.io/application/repository"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/messaging/emails"
	"github.com/hibiken/asynq"
)

var HandleWorkspaceInviteTaskName mq_types.Queues = "send_workspace_invite"

type WorkspaceInvitePayload struct {
	Email         string
	Permissions   []entities.MemberPermissions
	WorkspaceID   string
	InvitedBy     string
	WorkspaceName string
}

func HandleWorkspaceInviteTask(ctx context.Context, t *asynq.Task) error {
	var payload WorkspaceInvitePayload
	err := json.Unmarshal(t.Payload(), &payload)
	if err != nil {
		logger.Error("an error occured while unmarshalling workspace invite payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	inviteRepo := repository.WorkspaceInviteRepo()
	inviteExists, err := inviteRepo.CountDocs(map[string]interface{}{
		"workspaceID": payload.WorkspaceID,
		"email":       payload.Email,
	})
	if err != nil {
		logger.Error("an error occured while trying to check if invite exists", logger.LoggerOptions{
			Key:  "err",
			Data: err.Error(),
		}, logger.LoggerOptions{
			Key:  "invite",
			Data: payload,
		})
		return err
	}
	if inviteExists != 0 {
		return nil
	}
	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		Email:     &payload.Email,
		TokenType: auth.AccessToken,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour * 24 * 14).Unix(), //lasts for 14 days
	})
	if err != nil {
		logger.Error("an error occured while generating access token for workspace invite", logger.LoggerOptions{
			Key:  "err",
			Data: err.Error(),
		}, logger.LoggerOptions{
			Key:  "invite",
			Data: payload,
		})
		return err
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	cache.Cache.CreateEntry(fmt.Sprintf("%s-%s-invite-token", payload.Email, payload.WorkspaceID), hashedAccessToken, time.Hour*24*14)

	emails.EmailService.SendEmail(payload.Email, fmt.Sprintf("You have been invited to join %s", payload.WorkspaceName), "org_member_added", map[string]any{
		"ORG_NAME":   payload.WorkspaceName,
		"LOGIN_LINK": fmt.Sprintf("%s/workspace/invite/%s", os.Getenv("CLIENT_URL"), *accessToken),
	})

	_, err = inviteRepo.CreateOne(context.Background(), entities.WorkspaceInvite{
		Email:         payload.Email,
		WorkspaceID:   payload.WorkspaceID,
		Permissions:   payload.Permissions,
		InvitedByID:   payload.InvitedBy,
		WorkspaceName: payload.WorkspaceName,
	})
	if err != nil {
		logger.Error("an error occured while trying to create invite", logger.LoggerOptions{
			Key:  "err",
			Data: err.Error(),
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		})
		return err
	}
	return nil
}
