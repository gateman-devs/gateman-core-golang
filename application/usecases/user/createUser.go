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

func CreateUserUseCase(ctx any, payload *dto.CreateUserDTO, deviceID *string, userAgent *string, encryptedSecret *string, deviceName *string) (*string, *string, *uint, error) {
	var availability_filter = map[string]any{}
	var localNumber *string
	if payload.Email != nil {
		availability_filter["email"] = strings.ToLower(*payload.Email)
		payload.Phone = nil
	} else if payload.Phone != nil && payload.Phone.LocalNumber != "" {
		localNumber = &payload.Phone.LocalNumber
		availability_filter["phone.localNumber"] = payload.Phone.LocalNumber
		payload.Email = nil
	}
	userRepo := repository.UserRepo()
	account, err := userRepo.FindOneByFilter(availability_filter)
	if err != nil {
		apperrors.UnknownError(ctx, err, deviceID)
		return nil, nil, nil, err
	}
	if account != nil {
		if account.Image == "" {
			url, err := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", account.ID, "accountimage"), types.SignedURLPermission{
				Write: true,
			})
			if err != nil {
				logger.Error("an error occured while generating url for setting account image", logger.LoggerOptions{
					Key:  "error",
					Data: err,
				})
				apperrors.UnknownError(ctx, err, deviceID)
				return nil, nil, nil, nil
			}
			token, err := auth.GenerateAuthToken(auth.ClaimsData{
				UserID:    account.ID,
				UserAgent: account.UserAgent,
				Email:     payload.Email,
				PhoneNum:  localNumber,
				DeviceID:  *deviceID,
				Intent:    "face_verification",
				IssuedAt:  time.Now().Unix(),
				ExpiresAt: time.Now().Local().Add(time.Minute * 10).Unix(), //lasts for 10 mins
			})
			if err != nil {
				apperrors.UnknownError(ctx, err, deviceID)
				return nil, nil, nil, nil
			}
			hashedToken, _ := cryptography.CryptoHahser.HashString(*token, nil)
			cache.Cache.CreateEntry(account.ID, hashedToken, time.Minute*10) // token should last for 10 mins
			return token, url, &constants.ACCOUNT_EXISTS_UNVERIFIED, nil
		}
		if !account.VerifiedAccount {
			if account.Email != nil {
				otp, err := auth.GenerateOTP(6, *account.Email)
				if err != nil {
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}
				emails.EmailService.SendEmail(*account.Email, "Verify your AuthOne account", "authone_user_welcome", map[string]any{
					"OTP": otp,
				})
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *account.Email), "verify_account", time.Minute*10)
			} else {
				otp, err := auth.GenerateOTP(6, account.Phone.ISOCode)
				if err != nil {
					apperrors.FatalServerError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}
				ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber), false, otp)
				encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
				if err != nil {
					apperrors.UnknownError(ctx, err, deviceID)
					return nil, nil, nil, nil
				}
				cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", account.Phone.ISOCode), *encryptedRef, time.Minute*10)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", account.Phone.ISOCode), "verify_account", time.Minute*10)
			}
			return nil, nil, &constants.ACCOUNT_EXISTS_EMAIL_OR_PHONE_UNVERIFIED, nil
		}
		for i, device := range account.Devices {
			if &device.ID == deviceID {
				account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
				break
			}
		}
		account.Devices = append(account.Devices, entities.Device{
			ID:     *deviceID,
			Name:   *deviceName,
			Secret: *encryptedSecret,
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
			apperrors.UnknownError(ctx, err, deviceID)
			return nil, nil, nil, err
		}
		hashedDeviceID, _ := cryptography.CryptoHahser.HashString(*deviceID, []byte(""))
		cache.Cache.CreateEntry(string(hashedDeviceID), encryptedSecret, time.Minute*0)
		url, err := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", account.ID, *deviceID), types.SignedURLPermission{
			Write: true,
		})
		if err != nil {
			logger.Error("an error occured while generating url for device verification", logger.LoggerOptions{
				Key:  "error",
				Data: err,
			})
			apperrors.UnknownError(ctx, err, deviceID)
			return nil, nil, nil, err
		}
		return nil, url, &constants.ACCOUNT_EXISTS, nil
	}

	if os.Getenv("ENV") == "prod" {
		if payload.Email != nil {
			found := cache.Cache.FindOne(fmt.Sprintf("%s-email-blacklist", *payload.Email))
			if found != nil {
				err = fmt.Errorf(`email address "%s" has been flagged as unacceptable on our system`, *payload.Email)
				apperrors.ClientError(ctx, err.Error(), nil, nil, deviceID)
				return nil, nil, nil, err
			}
			result, err := polymercore.PolymerService.EmailStatus(*payload.Email)
			if err != nil {
				apperrors.ExternalDependencyError(ctx, "polymer-core", "500", err, deviceID)
				return nil, nil, nil, err
			}
			if !result {
				apperrors.ClientError(ctx, fmt.Sprintf(`email address "%s" has been flagged as unacceptable on our system`, *payload.Email), nil, nil, deviceID)
				cache.Cache.CreateEntry(fmt.Sprintf("%s-email-blacklist", *payload.Email), payload.Email, time.Minute*0)
				return nil, nil, nil, err
			}
		}
	}

	hashedPassword, _ := cryptography.CryptoHahser.HashString(*payload.Password, []byte("somefixedsaltvalue"))
	id := utils.GenerateUULDString()
	_, err = userRepo.CreateOne(context.TODO(), entities.User{
		ID:       id,
		Email:    payload.Email,
		Password: string(hashedPassword),
		Phone:    payload.Phone,
		Devices: []entities.Device{{
			ID:     *deviceID,
			Secret: *encryptedSecret,
			Name:   *deviceName,
		}},
		UserAgent: *userAgent,
		// Image:     fmt.Sprintf("%s/%s", id, "accountimage"),
	})
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(*deviceID, []byte("somefixedsaltvalue"))
	cache.Cache.CreateEntry(string(hashedDeviceID), *encryptedSecret, time.Minute*0)
	if err != nil {
		logger.Error("could not create user", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx, err, deviceID)
		return nil, nil, nil, err
	}
	url, err := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", id, "accountimage"), types.SignedURLPermission{
		Write: true,
	})
	if err != nil {
		logger.Error("an error occured while generating url for setting account image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx, err, deviceID)
		return nil, nil, nil, err
	}

	token, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:    id,
		UserAgent: *userAgent,
		Email:     payload.Email,
		PhoneNum:  localNumber,
		DeviceID:  *deviceID,
		Intent:    "face_verification",
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Local().Add(time.Minute * 10).Unix(), //lasts for 10 mins
	})
	if err != nil {
		apperrors.UnknownError(ctx, err, deviceID)
		return nil, nil, nil, nil
	}
	hashedToken, _ := cryptography.CryptoHahser.HashString(*token, nil)
	cache.Cache.CreateEntry(id, hashedToken, time.Minute*10) // token should last for 10 mins

	return token, url, &constants.ACCOUNT_CREATED, nil
}
