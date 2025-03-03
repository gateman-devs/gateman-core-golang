package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	user_usecases "gateman.io/application/usecases/user"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/biometric"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/logger"
	messagequeue "gateman.io/infrastructure/message_queue"
	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
	sms "gateman.io/infrastructure/messaging/sms"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func KeyExchange(ctx *interfaces.ApplicationContext[dto.KeyExchangeDTO]) {
	// serverPublicKey, _ := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.Body.ClientPublicKey)
	// if serverPublicKey == nil {
	// 	return
	// }
	// server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusCreated, "key exchanged", hex.EncodeToString(serverPublicKey), nil, nil)
}

func AuthenticateUser(ctx *interfaces.ApplicationContext[dto.CreateUserDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	if ctx.Body.Email == nil && ctx.Body.Phone == nil {
		apperrors.ClientError(ctx.Ctx, "One of email or phone is required", nil, nil)
		return
	}
	token, url, code, err := user_usecases.CreateUserUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent, ctx.DeviceName)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "authentication complete", map[string]any{
		"url":  url,
		"code": code,
	}, nil, nil, token, nil)
}

func VerifyUserAccount(ctx *interfaces.ApplicationContext[any]) {
	userRepo := repository.UserRepo()
	filter := map[string]any{}
	if ctx.GetStringContextData("OTPEmail") != "" {
		filter["email"] = ctx.GetStringContextData("OTPEmail")
	} else {
		filter["phone.localNumber"] = ctx.GetStringContextData("OTPPhone")
	}
	profile, err := userRepo.FindOneByFilter(filter, options.FindOne().SetProjection(map[string]any{
		"_id":       1,
		"firstName": 1,
		"lastName":  1,
		"deviceID":  1,
		"email":     1,
		"phone":     1,
	}))
	if err != nil {
		logger.Error("an error occured while fetching user profile for verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	if profile == nil {
		apperrors.NotFoundError(ctx.Ctx, "Account not found")
		return
	}
	success, err := userRepo.UpdatePartialByFilter(filter, map[string]any{
		"verifiedAccount": true,
	})
	if err != nil {
		logger.Error("an error occured while verifying org email", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	if !success {
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	var phone *string
	if profile.Phone != nil {
		phone = utils.GetStringPointer(fmt.Sprintf("%s%s", profile.Phone.Prefix, profile.Phone.LocalNumber))
	}
	token, err := auth.GenerateAuthToken(auth.ClaimsData{
		Email:           profile.Email,
		UserID:          profile.ID,
		UserAgent:       profile.UserAgent,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		Intent:          "face_verification",
		TokenType:       auth.AccessToken,
		VerifiedAccount: true,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	if err != nil {
		logger.Error("an error occured while generating auth token after org verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}

	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*token, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Minute*10)
	url, err := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", profile.ID, "accountimage"), types.SignedURLPermission{
		Write: true,
	})
	if err != nil {
		logger.Error("an error occured while generating url for setting account image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "email verified", url, nil, nil, token, nil)
}

func VerifyOTP(ctx *interfaces.ApplicationContext[dto.VerifyOTPDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	if ctx.Body.Email == nil && ctx.Body.Phone == nil {
		apperrors.ClientError(ctx.Ctx, "One of email or phone is required", nil, nil)
		return
	}
	var channel = ""
	var filter = map[string]any{}
	if ctx.Body.Email != nil {
		channel = *ctx.Body.Email
		filter["email"] = channel
		msg, success := auth.VerifyOTP(channel, ctx.Body.OTP)
		if !success {
			apperrors.ClientError(ctx.Ctx, msg, nil, nil)
			return
		}
	} else {
		channel = *ctx.Body.Phone
		filter["phone.localNumber"] = channel
		msg, success := auth.VerifyOTP(channel, ctx.Body.OTP)
		if !success {
			logger.Info("possible sms otp attempted to be verified as whatsapp otp", logger.LoggerOptions{
				Key:  "message",
				Data: msg,
			})
			otpRef := cache.Cache.FindOne(fmt.Sprintf("%s-sms-otp-ref", channel))
			if otpRef == nil {
				apperrors.NotFoundError(ctx.Ctx, "otp has expired")
				return
			}
			d, err := cryptography.DecryptData(*otpRef, nil)
			if err != nil {
				logger.Error("error dcrypting sms otp ref", logger.LoggerOptions{
					Key:  "ref",
					Data: *otpRef,
				}, logger.LoggerOptions{
					Key:  "channel",
					Data: channel,
				}, logger.LoggerOptions{
					Key:  "error",
					Data: err,
				})
				apperrors.NotFoundError(ctx.Ctx, "an error occured whil everifying otp")
				return
			}
			success := sms.SMSService.VerifyOTP(string(d), ctx.Body.OTP)
			if !success {
				apperrors.ClientError(ctx.Ctx, "wrong otp", nil, nil)
				return
			}
			cache.Cache.DeleteOne(fmt.Sprintf("%s-sms-otp-ref", channel))
		}
	}
	otpIntent := cache.Cache.FindOne(fmt.Sprintf("%s-otp-intent", channel))
	if otpIntent == nil {
		logger.Error("otp intent missing")
		apperrors.ClientError(ctx.Ctx, "otp expired", nil, nil)
		return
	}
	token, err := auth.GenerateAuthToken(auth.ClaimsData{
		Email:     ctx.Body.Email,
		PhoneNum:  ctx.Body.Phone,
		Intent:    *otpIntent,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Minute * time.Duration(10)).Unix(), //lasts for 10 mins
	})
	if err != nil {
		apperrors.FatalServerError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "otp verified", nil, nil, nil, token, nil)
}

func ResendOTP(ctx *interfaces.ApplicationContext[dto.ResendOTPDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	if ctx.Body.Email != nil {
		otp, err := auth.GenerateOTP(6, *ctx.Body.Email)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err)
			return
		}

		payload, err := json.Marshal(queue_tasks.EmailPayload{
			Opts: map[string]any{
				"OTP": otp,
			},
			To:       *ctx.Body.Email,
			Subject:  "Gateman OTP",
			Template: "authone_user_welcome",
			Intent:   "verify_account",
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
	}
	if ctx.Body.Phone != nil {
		otp, err := auth.GenerateOTP(6, *ctx.Body.Phone)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err)
			return
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", *ctx.Body.PhonePrefix, *ctx.Body.Phone), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", *ctx.Body.Phone), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *ctx.Body.Phone), "verify_account", time.Minute*10)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "otp sent", nil, nil, nil, nil, nil)
}

func VeirfyDeviceImage(ctx *interfaces.ApplicationContext[dto.VerifyDeviceDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	var accountSearchFilter = map[string]any{}
	if ctx.Body.Email != nil {
		accountSearchFilter["email"] = *ctx.Body.Email
	} else {
		accountSearchFilter["phone.localNumber"] = *ctx.Body.Phone
	}
	userRepo := repository.UserRepo()
	account, err := userRepo.FindOneByFilter(accountSearchFilter)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "account not found")
		return
	}
	if !account.VerifiedAccount {
		apperrors.AuthenticationError(ctx.Ctx, "Verify your account before attempting to login")
		return
	}
	exists, err := fileupload.FileUploader.CheckFileExists(fmt.Sprintf("%s/%s", account.ID, ctx.DeviceID))
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "azure", "500", err)
		return
	}
	if !exists {
		apperrors.ClientError(ctx.Ctx, "Image has not been uploaded. Request for a new url and upload image before attempting this request again.", nil, utils.GetUIntPointer(http.StatusBadRequest))
		return
	}
	url, _ := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), ctx.DeviceID), types.SignedURLPermission{
		Read: true,
	})
	alive, err := biometric.BiometricService.LivenessCheck(url)
	if err != nil {
		logger.Error("something went wrong when verifying image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	if !alive {
		apperrors.ClientError(ctx.Ctx, "Please make sure to take a clear picture of your face", nil, nil)
		return
	}
	accountImgURL, _ := fileupload.FileUploader.GeneratedSignedURL(account.Image, types.SignedURLPermission{
		Read: true,
	})
	match, err := biometric.BiometricService.FaceMatch(url, accountImgURL)
	if err != nil {
		logger.Error("something went wrong when match images", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	if !match {
		apperrors.ClientError(ctx.Ctx, "Face mismatch", nil, nil)
		return
	}
	var savedDevice entities.Device
	for i, device := range account.Devices {
		if device.ID == ctx.DeviceID {
			savedDevice = account.Devices[i]
			account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
			break
		}
	}
	account.Devices = append(account.Devices, entities.Device{
		ID:        savedDevice.ID,
		Name:      savedDevice.Name,
		LastLogin: savedDevice.LastLogin,
		Verified:  true,
	})
	_, err = userRepo.UpdatePartialByID(account.ID, map[string]any{
		"devices": account.Devices,
	})

	if err != nil {
		logger.Error("something went wrong when updating image status", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	var phone *string
	if account.Phone != nil {
		phone = utils.GetStringPointer(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber))
	}
	err = fileupload.FileUploader.DeleteFile(fmt.Sprintf("%s/%s", account.ID, ctx.DeviceID))
	if err != nil {
		logger.Error("an error occured while trying to clear user device image", logger.LoggerOptions{
			Key:  "filePath",
			Data: fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}

	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          account.ID,
		UserAgent:       account.UserAgent,
		Email:           account.Email,
		VerifiedAccount: account.VerifiedAccount,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	refreshToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          account.ID,
		UserAgent:       account.UserAgent,
		Email:           account.Email,
		VerifiedAccount: account.VerifiedAccount,
		TokenType:       auth.RefreshToken,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 180).Unix(), //lasts for 180 days
	})

	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)       // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 100 days
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "device verified", nil, nil, nil, accessToken, refreshToken)
}

func RefreshToken(ctx *interfaces.ApplicationContext[any]) {
	userRepo := repository.UserRepo()
	account, err := userRepo.FindByID(ctx.GetStringContextData("UserID"))
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "this account does not exist")
		return
	}
	var phone *string
	if account.Phone != nil {
		phone = utils.GetStringPointer(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber))
	}
	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          account.ID,
		UserAgent:       account.UserAgent,
		Email:           account.Email,
		VerifiedAccount: account.VerifiedAccount,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	refreshToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          account.ID,
		UserAgent:       account.UserAgent,
		Email:           account.Email,
		VerifiedAccount: account.VerifiedAccount,
		TokenType:       auth.RefreshToken,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 180).Unix(), //lasts for 180 days
	})

	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)       // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 100 days
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "token refreshed", nil, nil, nil, accessToken, refreshToken)
}
