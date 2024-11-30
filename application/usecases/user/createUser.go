package user_usecases

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/repository"
	polymercore "authone.usepolymer.co/application/services/polymer-core"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	fileupload "authone.usepolymer.co/infrastructure/file_upload"
	"authone.usepolymer.co/infrastructure/file_upload/types"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/messaging/emails"
	"authone.usepolymer.co/infrastructure/messaging/sms"
)

func CreateUserUseCase(ctx any, payload *dto.CreateUserDTO, deviceID *string, userAgent *string, deviceName *string) (*string, *string, *uint, error) {
	var availabilityFilter = map[string]any{}
	if payload.Email != nil {
		availabilityFilter["email"] = strings.ToLower(*payload.Email)
		payload.Phone = nil
	} else if payload.Phone != nil && payload.Phone.LocalNumber != "" {
		availabilityFilter["phone.localNumber"] = payload.Phone.LocalNumber
		payload.Email = nil
	}
	fmt.Println(availabilityFilter, payload.Phone)
	userRepo := repository.UserRepo()
	account, err := userRepo.FindOneByFilter(availabilityFilter)
	if err != nil {
		apperrors.UnknownError(ctx, err)
		return nil, nil, nil, err
	}
	if account != nil {
		if !account.VerifiedAccount {
			if account.Email != nil {
				otp, err := auth.GenerateOTP(6, *account.Email)
				if err != nil {
					apperrors.FatalServerError(ctx, err)
					return nil, nil, nil, nil
				}
				emails.EmailService.SendEmail(*account.Email, "Verify your AuthOne account", "authone_user_welcome", map[string]any{
					"OTP": otp,
				})
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *account.Email), "verify_account", time.Minute*10)
			} else {
				otp, err := auth.GenerateOTP(6, account.Phone.LocalNumber)
				if err != nil {
					apperrors.FatalServerError(ctx, err)
					return nil, nil, nil, nil
				}
				ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber), false, otp)
				encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
				if err != nil {
					apperrors.UnknownError(ctx, err)
					return nil, nil, nil, nil
				}
				cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", account.Phone.LocalNumber), *encryptedRef, time.Minute*10)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", account.Phone.LocalNumber), "verify_account", time.Minute*10)
			}

			for i, device := range account.Devices {
				if device.ID == *deviceID {
					account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
					break
				}
			}
			account.Devices = append(account.Devices, entities.Device{
				ID:        *deviceID,
				Name:      *deviceName,
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
				apperrors.UnknownError(ctx, err)
				return nil, nil, nil, err
			}
			return nil, nil, &constants.ACCOUNT_EXISTS_EMAIL_OR_PHONE_UNVERIFIED, nil
		}
		if account.Image == "" {
			if account.Email != nil {
				otp, err := auth.GenerateOTP(6, *account.Email)
				if err != nil {
					apperrors.FatalServerError(ctx, err)
					return nil, nil, nil, nil
				}
				emails.EmailService.SendEmail(*account.Email, "Verify your gateman login", "authone_user_welcome", map[string]any{
					"OTP": otp,
				})
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *account.Email), "verify_account", time.Minute*10)
			} else {
				otp, err := auth.GenerateOTP(6, account.Phone.LocalNumber)
				if err != nil {
					apperrors.FatalServerError(ctx, err)
					return nil, nil, nil, nil
				}
				ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber), false, otp)
				encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
				if err != nil {
					apperrors.UnknownError(ctx, err)
					return nil, nil, nil, nil
				}
				cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", account.Phone.LocalNumber), *encryptedRef, time.Minute*10)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", account.Phone.LocalNumber), "verify_account", time.Minute*10)
			}

			for i, device := range account.Devices {
				if device.ID == *deviceID {
					account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
					break
				}
			}
			account.Devices = append(account.Devices, entities.Device{
				ID:        *deviceID,
				Name:      *deviceName,
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
				apperrors.UnknownError(ctx, err)
				return nil, nil, nil, err
			}
			return nil, nil, &constants.ACCOUNT_EXISTS_UNVERIFIED, nil
		}
		for i, device := range account.Devices {
			if device.ID == *deviceID {
				account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
				break
			}
		}
		account.Devices = append(account.Devices, entities.Device{
			ID:        *deviceID,
			Name:      *deviceName,
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
			apperrors.UnknownError(ctx, err)
			return nil, nil, nil, err
		}
		url, err := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", account.ID, *deviceID), types.SignedURLPermission{
			Write: true,
		})
		if err != nil {
			logger.Error("an error occured while generating url for device verification", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			apperrors.UnknownError(ctx, err)
			return nil, nil, nil, err
		}
		return nil, url, &constants.ACCOUNT_EXISTS, nil
	}

	if os.Getenv("ENV") == "prod" {
		if payload.Email != nil {
			found := cache.Cache.FindOne(fmt.Sprintf("%s-email-blacklist", *payload.Email))
			if found != nil {
				err = fmt.Errorf(`email address "%s" has been flagged as unacceptable on our system`, *payload.Email)
				apperrors.ClientError(ctx, err.Error(), nil, nil)
				return nil, nil, nil, err
			}
			result, err := polymercore.PolymerService.EmailStatus(*payload.Email)
			if err != nil {
				apperrors.ExternalDependencyError(ctx, "polymer-core", "500", err)
				return nil, nil, nil, err
			}
			if !result {
				apperrors.ClientError(ctx, fmt.Sprintf(`email address "%s" has been flagged as unacceptable on our system`, *payload.Email), nil, nil)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-email-blacklist", *payload.Email), payload.Email, time.Minute*0)
				return nil, nil, nil, err
			}
		}
	}

	id := utils.GenerateUULDString()
	_, err = userRepo.CreateOne(context.TODO(), entities.User{
		ID:    id,
		Email: payload.Email,
		Phone: payload.Phone,
		Devices: []entities.Device{{
			ID:        *deviceID,
			Name:      *deviceName,
			LastLogin: time.Now(),
		}},
		UserAgent: *userAgent,
		// Image:     fmt.Sprintf("%s/%s", id, "accountimage"),
	})
	if err != nil {
		logger.Error("could not create user", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx, err)
		return nil, nil, nil, err
	}

	if payload.Email != nil {
		otp, err := auth.GenerateOTP(6, *payload.Email)
		if err != nil {
			apperrors.FatalServerError(ctx, err)
			return nil, nil, nil, nil
		}
		emails.EmailService.SendEmail(*payload.Email, "Verify your AuthOne account", "authone_user_welcome", map[string]any{
			"OTP": otp,
		})
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *payload.Email), "verify_account", time.Minute*10)
	} else {
		otp, err := auth.GenerateOTP(6, payload.Phone.LocalNumber)
		if err != nil {
			apperrors.FatalServerError(ctx, err)
			return nil, nil, nil, nil
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", payload.Phone.Prefix, payload.Phone.LocalNumber), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx, err)
			return nil, nil, nil, nil
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", payload.Phone.LocalNumber), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", payload.Phone.LocalNumber), "verify_account", time.Minute*10)
	}
	return nil, nil, &constants.ACCOUNT_CREATED, nil
}
