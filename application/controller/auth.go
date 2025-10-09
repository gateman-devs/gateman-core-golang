package controller

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	auth_usecases "gateman.io/application/usecases/auth"
	user_usecases "gateman.io/application/usecases/user"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/biometric"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/ipresolver"
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
	serverPublicKey, _, _ := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.DeviceID, ctx.Body.ClientPublicKey)
	if serverPublicKey == nil {
		return
	}
	server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusCreated, "key exchanged", hex.EncodeToString(serverPublicKey), nil, nil)
}

func AuthenticateUser(ctx *interfaces.ApplicationContext[dto.CreateUserDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if ctx.Body.Email == nil && ctx.Body.Phone == nil {
		apperrors.ClientError(ctx.Ctx, "One of email or phone is required", nil, nil, ctx.DeviceID)
		return
	}
	token, url, code, err := user_usecases.CreateUserUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent, ctx.DeviceName)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "authentication complete", map[string]any{
		"url":         url,
		"code":        code,
		"accessToken": token,
	}, nil, nil, &ctx.DeviceID)
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
		apperrors.NotFoundError(ctx.Ctx, "Account not found", &ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 10).Unix(), //lasts for 10 days
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
	}, time.Minute*10)
	if err != nil {
		logger.Error("an error occured while generating url for setting account image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "email verified", map[string]any{
		"url":         url,
		"accessToken": token,
	}, nil, nil, &ctx.DeviceID)
}

func VerifyWorkspaceAccount(ctx *interfaces.ApplicationContext[any]) {
	workspaceRepo := repository.WorkspaceRepository()
	workspace, err := workspaceRepo.FindOneByFilter(map[string]interface{}{
		"email": ctx.GetStringContextData("OTPEmail"),
	}, options.FindOne().SetProjection(map[string]any{
		"_id":   1,
		"email": 1,
	}))
	if err != nil {
		logger.Error("an error occured while fetching workspace for verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	if workspace == nil {
		apperrors.NotFoundError(ctx.Ctx, "Workspace not found", &ctx.DeviceID)
		return
	}
	if workspace.VerifiedEmail {
		apperrors.ClientError(ctx.Ctx, "Workspace already verified", nil, nil, ctx.DeviceID)
		return
	}
	success, err := workspaceRepo.UpdatePartialByFilter(map[string]interface{}{
		"email": ctx.GetStringContextData("OTPEmail"),
	}, map[string]any{
		"verifiedEmail": true,
	})
	if err != nil {
		logger.Error("an error occured while verifying org email", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	if !success {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	workspaceMemberRepo := repository.WorkspaceMemberRepo()

	superAdmin, _ := workspaceMemberRepo.FindOneByFilter(map[string]interface{}{
		"email": ctx.GetStringContextData("OTPEmail"),
	})

	var savedDevice *entities.Device
	for i, device := range superAdmin.Devices {
		if device.ID == ctx.DeviceID {
			savedDevice = &superAdmin.Devices[i]
			superAdmin.Devices = append(superAdmin.Devices[:i], superAdmin.Devices[i+1:]...)
			break
		}
	}
	if savedDevice == nil {
		ipLookupRes, _ := ipresolver.IPResolverInstance.LookUp(ctx.Param["ip"].(string))
		superAdmin.Devices = append(superAdmin.Devices, entities.Device{
			ID:                ctx.DeviceID,
			Name:              ctx.DeviceName,
			LastLogin:         time.Now(),
			LastLoginLocation: fmt.Sprintf("%s, %s - (%f, %f)", strings.ToUpper(ipLookupRes.City), strings.ToUpper(ipLookupRes.CountryCode), ipLookupRes.Longitude, ipLookupRes.Latitude),
			Verified:          true,
		})
	} else {
		ipLookupRes, _ := ipresolver.IPResolverInstance.LookUp(ctx.Param["ip"].(string))
		superAdmin.Devices = append(superAdmin.Devices, entities.Device{
			ID:                savedDevice.ID,
			Name:              savedDevice.Name,
			LastLogin:         time.Now(),
			LastLoginLocation: fmt.Sprintf("%s, %s - (%f, %f)", strings.ToUpper(ipLookupRes.City), strings.ToUpper(ipLookupRes.CountryCode), ipLookupRes.Longitude, ipLookupRes.Latitude),
			Verified:          true,
		})
	}
	workspaceMemberRepo.UpdatePartialByFilter(map[string]interface{}{
		"email": ctx.GetStringContextData("OTPEmail"),
	}, map[string]any{
		"verifiedEmail": true,
		"devices":       superAdmin.Devices,
	})
	if err != nil {
		logger.Error("an error occured while verifying workspace member email", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	if !success {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	superAdmin, _ = workspaceMemberRepo.FindOneByFilter(map[string]interface{}{
		"email": ctx.GetStringContextData("OTPEmail"),
	})

	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          superAdmin.ID,
		UserAgent:       superAdmin.UserAgent,
		Email:           &superAdmin.Email,
		VerifiedAccount: superAdmin.VerifiedEmail,
		WorkspaceID:     &workspace.ID,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	refreshToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          superAdmin.ID,
		UserAgent:       superAdmin.UserAgent,
		Email:           &superAdmin.Email,
		VerifiedAccount: superAdmin.VerifiedEmail,
		TokenType:       auth.RefreshToken,
		WorkspaceID:     &workspace.ID,
		DeviceID:        ctx.DeviceID,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 180).Unix(), //lasts for 180 days
	})

	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-workspace-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)       // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-workspace-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 100 days
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "email verified", map[string]any{
		"workspaceAccessToken":  accessToken,
		"workspaceRefreshToken": refreshToken,
		"profile":               superAdmin,
	}, nil, nil, &ctx.DeviceID)
}

func VerifyOTP(ctx *interfaces.ApplicationContext[dto.VerifyOTPDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if ctx.Body.Email == nil && ctx.Body.Phone == nil {
		apperrors.ClientError(ctx.Ctx, "One of email or phone is required", nil, nil, ctx.DeviceID)
		return
	}
	var channel = ""
	var filter = map[string]any{}
	if ctx.Body.Email != nil {
		channel = *ctx.Body.Email
		filter["email"] = channel
		msg, success := auth.VerifyOTP(channel, ctx.Body.OTP)
		if !success {
			apperrors.ClientError(ctx.Ctx, msg, nil, nil, ctx.DeviceID)
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
				apperrors.NotFoundError(ctx.Ctx, "otp has expired", &ctx.DeviceID)
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
				apperrors.NotFoundError(ctx.Ctx, "an error occured whil everifying otp", &ctx.DeviceID)
				return
			}
			success := sms.SMSService.VerifyOTP(string(d), ctx.Body.OTP)
			if !success {
				apperrors.ClientError(ctx.Ctx, "wrong otp", nil, nil, ctx.DeviceID)
				return
			}
			cache.Cache.DeleteOne(fmt.Sprintf("%s-sms-otp-ref", channel))
		}
	}
	otpIntent := cache.Cache.FindOne(fmt.Sprintf("%s-otp-intent", channel))
	if otpIntent == nil {
		logger.Error("otp intent missing")
		apperrors.ClientError(ctx.Ctx, "otp expired", nil, nil, ctx.DeviceID)
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
		apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "otp verified", map[string]any{
		"accessToken": token,
	}, nil, nil, &ctx.DeviceID)
}

func ResendOTP(ctx *interfaces.ApplicationContext[dto.ResendOTPDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if ctx.Body.Email != nil {
		otp, err := auth.GenerateOTP(6, *ctx.Body.Email)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
			return
		}

		payload, err := json.Marshal(queue_tasks.EmailPayload{
			Opts: map[string]any{
				"OTP":            otp,
				"EXPIRY_MINUTES": 10,
				"REQUEST_ACTION": "verify account",
				"APP_NAME":       "Gateman",
			},
			To:       *ctx.Body.Email,
			Subject:  "Gateman OTP",
			Template: "otp-request",
			Intent:   "verify_account",
		})
		if err != nil {
			logger.Error("error marshalling payload for email queue")
			apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
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
			apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
			return
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", *ctx.Body.PhonePrefix, *ctx.Body.Phone), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", *ctx.Body.Phone), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *ctx.Body.Phone), "verify_account", time.Minute*10)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "otp sent", nil, nil, nil, &ctx.DeviceID)
}

func VeirfyDeviceImage(ctx *interfaces.ApplicationContext[dto.VerifyDeviceDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "account not found", &ctx.DeviceID)
		return
	}
	if !account.VerifiedAccount {
		apperrors.AuthenticationError(ctx.Ctx, "Verify your account before attempting to login", ctx.DeviceID)
		return
	}
	exists, err := fileupload.FileUploader.CheckFileExists(fmt.Sprintf("%s/%s", account.ID, ctx.DeviceID))
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "azure", "500", err, ctx.DeviceID)
		return
	}
	if !exists {
		apperrors.ClientError(ctx.Ctx, "Image has not been uploaded. Request for a new url and upload image before attempting this request again.", nil, utils.GetUIntPointer(http.StatusBadRequest), ctx.DeviceID)
		return
	}
	url, _ := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), ctx.DeviceID), types.SignedURLPermission{
		Read: true,
	}, time.Minute*1)
	alive, err := biometric.BiometricService.ImageLivenessCheck(url)
	if err != nil {
		logger.Error("something went wrong when verifying image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if !alive.Success {
		apperrors.ClientError(ctx.Ctx, "Please make sure to take a clear picture of your face", nil, nil, ctx.DeviceID)
		return
	}
	accountImgURL, _ := fileupload.FileUploader.GeneratedSignedURL(account.Image, types.SignedURLPermission{
		Read: true,
	}, time.Minute*1)
	match, err := biometric.BiometricService.CompareFaces(url, accountImgURL)
	if err != nil {
		logger.Error("something went wrong when match images", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if !match.Success {
		apperrors.ClientError(ctx.Ctx, "Face mismatch", nil, nil, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*1)        // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 10 mins
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "device verified", map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}, nil, nil, &ctx.DeviceID)
}

func RefreshToken(ctx *interfaces.ApplicationContext[any]) {
	userRepo := repository.UserRepo()
	account, err := userRepo.FindByID(ctx.GetStringContextData("UserID"))
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "this account does not exist", &ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)       // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 100 days
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "token refreshed", map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}, nil, nil, &ctx.DeviceID)
}

func WorkspaceRefreshToken(ctx *interfaces.ApplicationContext[any]) {
	userRepo := repository.WorkspaceMemberRepo()
	account, err := userRepo.FindByID(ctx.GetStringContextData("UserID"))
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "this account does not exist", &ctx.DeviceID)
		return
	}
	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          account.ID,
		UserAgent:       account.UserAgent,
		Email:           &account.Email,
		VerifiedAccount: account.VerifiedEmail,
		DeviceID:        ctx.DeviceID,
		WorkspaceID:     &account.WorkspaceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	refreshToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          account.ID,
		UserAgent:       account.UserAgent,
		Email:           &account.Email,
		VerifiedAccount: account.VerifiedEmail,
		WorkspaceID:     &account.WorkspaceID,
		TokenType:       auth.RefreshToken,
		DeviceID:        ctx.DeviceID,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 180).Unix(), //lasts for 180 days
	})

	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-workspace-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)       // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-workspace-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 100 days
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "token refreshed", map[string]any{
		"workspaceAccessToken":  accessToken,
		"workspaceRefreshToken": refreshToken,
	}, nil, nil, &ctx.DeviceID)
}
