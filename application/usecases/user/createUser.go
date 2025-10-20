package user_usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/constants"
	"gateman.io/application/controller/dto"
	"gateman.io/application/repository"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/messaging/sms"
)

func CreateUserUseCase(ctx any, payload *dto.CreateUserDTO, deviceID string, userAgent string, deviceName string) (*string, *uint, error) {
	var availabilityFilter = map[string]any{}
	if payload.Email != nil {
		availabilityFilter["email"] = strings.ToLower(*payload.Email)
		payload.Phone = nil
	} else if payload.Phone != nil && payload.Phone.LocalNumber != "" {
		availabilityFilter["phone.localNumber"] = payload.Phone.LocalNumber
		payload.Email = nil
	}
	userRepo := repository.UserRepo()
	account, err := userRepo.FindOneByFilter(availabilityFilter)
	if err != nil {
		apperrors.UnknownError(ctx, err, nil, deviceID)
		return nil, nil, err
	}
	if account == nil {
		id := utils.GenerateUULDString()
		_, err = userRepo.CreateOne(context.TODO(), entities.User{
			ID:    id,
			Email: payload.Email,
			Phone: payload.Phone,
			Devices: []entities.Device{{
				ID:        deviceID,
				Name:      deviceName,
				LastLogin: time.Now(),
			}},
			UserAgent: userAgent,
			// Image:     fmt.Sprintf("%s/%s", id, "accountimage"),
		})
		if err != nil {
			logger.Error("could not create user", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			apperrors.UnknownError(ctx, err, nil, deviceID)
			return nil, nil, err
		}
	}

	if payload.Email != nil {
		otp, err := auth.GenerateOTP(6, *payload.Email)
		if err != nil {
			apperrors.FatalServerError(ctx, err, deviceID)
			return nil, nil, nil
		}
		emailPayload, err := json.Marshal(queue_tasks.EmailPayload{
			Opts: map[string]any{
				"OTP":            otp,
				"EXPIRY_MINUTES": 10,
				"REQUEST_ACTION": "verify account",
				"APP_NAME":       "Gateman",
			},
			To:       *payload.Email,
			Subject:  "Gateman OTP",
			Template: "otp-request",
			Intent:   ("verify_login"),
		})
		if err != nil {
			logger.Error("error marshalling payload for email queue")
			apperrors.FatalServerError(ctx, err, deviceID)
			return nil, nil, err
		}
		messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
			Payload:   emailPayload,
			Name:      queue_tasks.HandleEmailDeliveryTaskName,
			Priority:  mq_types.High,
			ProcessIn: 1,
		})
	} else {
		otp, err := auth.GenerateOTP(6, payload.Phone.LocalNumber)
		if err != nil {
			apperrors.FatalServerError(ctx, err, deviceID)
			return nil, nil, nil
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", payload.Phone.Prefix, payload.Phone.LocalNumber), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx, err, nil, deviceID)
			return nil, nil, nil
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", payload.Phone.LocalNumber), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", payload.Phone.LocalNumber), "verify_login", time.Minute*10)
	}
	return nil, &constants.ACCOUNT_CREATED, nil
}
