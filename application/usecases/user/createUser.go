package user_usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
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
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/messaging/sms"
)

func CreateUserUseCase(ctx any, payload *dto.CreateUserDTO, deviceID string, userAgent string, deviceName string) (*string, *string, *uint, error) {
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
		return nil, nil, nil, err
	}
	if account != nil {
		if !account.VerifiedAccount {
			if account.Email != nil {
				otp, err := auth.GenerateOTP(6, *account.Email)
				if err != nil {
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}

				emailPayload, err := json.Marshal(queue_tasks.EmailPayload{
					Opts: map[string]any{
						"OTP":            otp,
						"EXPIRY_MINUTES": 10,
						"REQUEST_ACTION": "verify account",
						"APP_NAME":       "Gateman",
					},
					To:       *payload.Email,
					Subject:  "Verify Your Gateman Email",
					Template: "otp-request",
					Intent:   ("verify_account"),
				})
				if err != nil {
					logger.Error("error marshalling payload for email queue")
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, err
				}
				messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
					Payload:   emailPayload,
					Name:      queue_tasks.HandleEmailDeliveryTaskName,
					Priority:  mq_types.High,
					ProcessIn: 1,
				})

			} else {
				otp, err := auth.GenerateOTP(6, account.Phone.LocalNumber)
				if err != nil {
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}
				ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber), false, otp)
				encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
				if err != nil {
					apperrors.UnknownError(ctx, err, nil, deviceID)
					return nil, nil, nil, nil
				}
				cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", account.Phone.LocalNumber), *encryptedRef, time.Minute*10)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", account.Phone.LocalNumber), "verify_account", time.Minute*10)
			}

			for i, device := range account.Devices {
				if device.ID == deviceID {
					account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
					break
				}
			}
			account.Devices = append(account.Devices, entities.Device{
				ID:        deviceID,
				Name:      deviceName,
				LastLogin: time.Now(),
			})
			_, err := userRepo.UpdateByID(account.ID, account)
			if err != nil {
				logger.Error("could not add new device", logger.LoggerOptions{
					Key:  "error",
					Data: err,
				}, logger.LoggerOptions{
					Key:  "devices",
					Data: account.Devices,
				})
				apperrors.UnknownError(ctx, err, nil, deviceID)
				return nil, nil, nil, err
			}
			return nil, nil, &constants.ACCOUNT_EXISTS_EMAIL_OR_PHONE_UNVERIFIED, nil
		}
		if account.Image == "" {
			if account.Email != nil {
				otp, err := auth.GenerateOTP(6, *account.Email)
				if err != nil {
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}

				payload, err := json.Marshal(queue_tasks.EmailPayload{
					Opts: map[string]any{
						"OTP":            otp,
						"EXPIRY_MINUTES": 10,
						"REQUEST_ACTION": "verify account",
						"APP_NAME":       "Gateman",
					},
					To:       *account.Email,
					Subject:  "Verify your gateman login",
					Template: "otp-request",
					Intent:   "verify_account",
				})
				if err != nil {
					logger.Error("error marshalling payload for email queue")
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, err
				}
				messagequeue.TaskQueue.Enqueue(mq_types.QueueTask{
					Payload:   payload,
					Name:      queue_tasks.HandleEmailDeliveryTaskName,
					Priority:  mq_types.High,
					ProcessIn: 1,
				})
			} else {
				otp, err := auth.GenerateOTP(6, account.Phone.LocalNumber)
				if err != nil {
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}
				ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber), false, otp)
				encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
				if err != nil {
					apperrors.UnknownError(ctx, err, nil, deviceID)
					return nil, nil, nil, nil
				}
				cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", account.Phone.LocalNumber), *encryptedRef, time.Minute*10)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", account.Phone.LocalNumber), "verify_account", time.Minute*10)
			}

			for i, device := range account.Devices {
				if device.ID == deviceID {
					account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
					break
				}
			}
			account.Devices = append(account.Devices, entities.Device{
				ID:        deviceID,
				Name:      deviceName,
				LastLogin: time.Now(),
			})
			_, err := userRepo.UpdateByID(account.ID, account)
			if err != nil {
				logger.Error("could not add new device", logger.LoggerOptions{
					Key:  "error",
					Data: err,
				}, logger.LoggerOptions{
					Key:  "devices",
					Data: account.Devices,
				})
				apperrors.UnknownError(ctx, err, nil, deviceID)
				return nil, nil, nil, err
			}
			return nil, nil, &constants.ACCOUNT_EXISTS_UNVERIFIED, nil
		}
		for i, device := range account.Devices {
			if device.ID == deviceID {
				account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
				break
			}
		}
		account.Devices = append(account.Devices, entities.Device{
			ID:        deviceID,
			Name:      deviceName,
			LastLogin: time.Now(),
		})
		_, err := userRepo.UpdateByID(account.ID, account)
		if err != nil {
			logger.Error("could not add new device", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			}, logger.LoggerOptions{
				Key:  "devices",
				Data: account.Devices,
			})
			apperrors.UnknownError(ctx, err, nil, deviceID)
			return nil, nil, nil, err
		}
		url, err := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", account.ID, deviceID), types.SignedURLPermission{
			Write: true,
		}, time.Minute*10)
		if err != nil {
			logger.Error("an error occured while generating url for device verification", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			apperrors.UnknownError(ctx, err, nil, deviceID)
			return nil, nil, nil, err
		}
		return nil, url, &constants.ACCOUNT_EXISTS, nil
	}

	if os.Getenv("APP_ENV") == "production" {
		if payload.Email != nil {
			// found := cache.Cache.FindOne(fmt.Sprintf("%s-email-blacklist", *payload.Email))
			// if found != nil {
			// 	err = fmt.Errorf(`email address "%s" has been flagged as unacceptable on our system`, *payload.Email)
			// 	apperrors.ClientError(ctx, err.Error(), nil, nil)
			// 	return nil, nil, nil, err
			// }
			// if err != nil {
			// 	apperrors.ExternalDependencyError(ctx, "polymer-core", "500", err)
			// 	return nil, nil, nil, err
			// }
			// if !result {
			// 	apperrors.ClientError(ctx, fmt.Sprintf(`email address "%s" has been flagged as unacceptable on our system`, *payload.Email), nil, nil)
			// 	cache.Cache.CreateEntry(fmt.Sprintf("%s-email-blacklist", *payload.Email), payload.Email, time.Minute*0)
			// 	return nil, nil, nil, err
			// }
		}
	}

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
		return nil, nil, nil, err
	}

	if payload.Email != nil {
		otp, err := auth.GenerateOTP(6, *payload.Email)
		if err != nil {
			apperrors.FatalServerError(ctx, err, deviceID)
			return nil, nil, nil, nil
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
			Intent:   ("verify_account"),
		})
		if err != nil {
			logger.Error("error marshalling payload for email queue")
			apperrors.FatalServerError(ctx, err, deviceID)
			return nil, nil, nil, err
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
			return nil, nil, nil, nil
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", payload.Phone.Prefix, payload.Phone.LocalNumber), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx, err, nil, deviceID)
			return nil, nil, nil, nil
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", payload.Phone.LocalNumber), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", payload.Phone.LocalNumber), "verify_account", time.Minute*10)
	}
	return nil, nil, &constants.ACCOUNT_CREATED, nil
}
