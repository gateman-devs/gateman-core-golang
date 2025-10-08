package workspace_usecases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/repository"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/ipresolver"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"go.mongodb.org/mongo-driver/mongo"
)

func CreateWorkspaceUseCase(ctx any, payload *dto.CreateWorkspaceDTO, deviceID string, deviceName string, userAgent string, ip string) error {
	workspaceMemberRepo := repository.WorkspaceMemberRepo()
	workspaceRepo := repository.WorkspaceRepository()
	existingWorkspace, _ := workspaceRepo.FindOneByFilter(map[string]interface{}{
		"email": payload.Email,
	})
	if existingWorkspace != nil {

		apperrors.ClientError(ctx, "this email is already attached to a workspace", nil, nil, deviceID)
		return errors.New("this email is already attached to a workspace")
	}
	err := workspaceRepo.StartTransaction(func(sc mongo.Session, c context.Context) error {
		workspaceID := utils.GenerateUULDString()
		hashedPassword, err := cryptography.CryptoHahser.HashString(payload.Password, nil)
		if err != nil {
			payload.Password = ""
			logger.Error("an error occured while trying to hash workspace member password", logger.LoggerOptions{
				Key:  "payload",
				Data: payload,
			})
			return errors.New("an error occured while trying to hash workspace member password")
		}

		ipLookupRes, _ := ipresolver.IPResolverInstance.LookUp(ip)
		orgMember := entities.WorkspaceMember{
			Permissions:   []entities.MemberPermissions{entities.SUPER_ACCESS},
			ID:            utils.GenerateUULDString(),
			WorkspaceID:   workspaceID,
			WorkspaceName: payload.Name,
			Password:      string(hashedPassword),
			Email:         payload.Email,
			FirstName:     "Super",
			LastName:      "Administrator",
			UserAgent:     userAgent,
			Devices: []entities.Device{
				{
					Name:              deviceName,
					ID:                deviceID,
					Verified:          false,
					LastLogin:         time.Now(),
					LastLoginLocation: fmt.Sprintf("%s, %s - (%f, %f)", strings.ToUpper(ipLookupRes.City), strings.ToUpper(ipLookupRes.CountryCode), ipLookupRes.Longitude, ipLookupRes.Latitude),
				},
			},
		}

		orgData := entities.Workspace{
			Name:          payload.Name,
			Sector:        payload.Sector,
			Country:       payload.Country,
			SuperMember:   orgMember.ID,
			ID:            workspaceID,
			Email:         payload.Email,
			VerifiedEmail: false,
		}
		_, trxErr := workspaceRepo.CreateOne(context.TODO(), orgData)
		if trxErr != nil {
			logger.Error("an error occured while creating an org", logger.LoggerOptions{
				Key:  "error",
				Data: trxErr,
			}, logger.LoggerOptions{
				Key:  "payload",
				Data: orgData,
			})
			sc.AbortTransaction(c)
			return trxErr
		}

		_, trxErr = workspaceMemberRepo.CreateOne(context.TODO(), orgMember)
		if trxErr != nil {
			logger.Error("an error occured while creating an org member", logger.LoggerOptions{
				Key:  "error",
				Data: trxErr,
			}, logger.LoggerOptions{
				Key:  "payload",
				Data: orgMember,
			})
			sc.AbortTransaction(c)
			return trxErr
		}

		(sc).CommitTransaction(c)
		return nil
	})

	if err != nil {
		apperrors.UnknownError(ctx, err, nil, deviceID)
		return err
	}

	otp, err := auth.GenerateOTP(6, payload.Email)
	if err != nil {
		apperrors.FatalServerError(ctx, err, deviceID)
		return err
	}
	emailPayload, err := json.Marshal(queue_tasks.EmailPayload{
		Opts: map[string]any{
			"OTP_CODE":                otp,
			"WORKSPACE_NAME":          payload.Name,
			"WORKSPACE_COUNTRY":       payload.Country,
			"WORKSPACE_SECTOR":        payload.Sector,
			"MANUAL_VERIFICATION_URL": "https://app.gateman.io/workspace/manual-verification",
			"EXPIRY_MINUTES":          "10",
			"SUPPORT_URL":             "https://support.gateman.io", // Example
			"STATUS_URL":              "https://status.gateman.io", // Example
			"PRIVACY_URL":             "https://gateman.io/privacy", // Example
		},
		To:       payload.Email,
		Subject:  "Gateman OTP",
		Template: "workspace_created",
		Intent:   "verify_workspace",
	})
	if err != nil {
		logger.Error("error marshalling payload for email queue")
		apperrors.FatalServerError(ctx, err, deviceID)
		return err
	}
	messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
		Payload:   emailPayload,
		Name:      queue_tasks.HandleEmailDeliveryTaskName,
		Priority:  mq_types.High,
		ProcessIn: 1,
	})
	cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", payload.Email), "verify_workspace", time.Minute*10)
	return nil
}
