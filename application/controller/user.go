package controller

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/biometric"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	fileupload "authone.usepolymer.co/infrastructure/file_upload"
	"authone.usepolymer.co/infrastructure/file_upload/types"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/messaging/emails"
	sms "authone.usepolymer.co/infrastructure/messaging/sms"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
)

func SetAccountImage(ctx *interfaces.ApplicationContext[any]) {
	exists, err := fileupload.FileUploader.CheckFileExists(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"))
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "azure", "500", err, ctx.DeviceID)
		return
	}
	if !exists {
		apperrors.ClientError(ctx.Ctx, "Image has not been uploaded. Request for a new url and upload image before attempting this request again.", nil, utils.GetUIntPointer(http.StatusBadRequest), ctx.DeviceID)
		return
	}
	url, _ := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"), types.SignedURLPermission{
		Read: true,
	})
	alive, err := biometric.BiometricService.LivenessCheck(url)
	if err != nil {
		logger.Error("something went wrong when verifying image", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if !alive {
		apperrors.ClientError(ctx.Ctx, "Please make sure to take a clear picture of your face", nil, nil, ctx.DeviceID)
		return
	}
	var availability_filter = map[string]any{}
	if ctx.GetStringContextData("Email") != "" {
		availability_filter["email"] = strings.ToLower(ctx.GetStringContextData("Email"))
	} else if ctx.GetStringContextData("Phone") != "" {
		availability_filter["phone.localNumber"] = ctx.GetStringContextData("Phone")
	}
	userRepo := repository.UserRepo()
	account, err := userRepo.FindOneByFilter(availability_filter)
	if err != nil {
		apperrors.UnknownError(ctx, err, ctx.DeviceID)
		return
	}
	if account.Email != nil {
		otp, err := auth.GenerateOTP(6, *account.Email)
		if err != nil {
			apperrors.FatalServerError(ctx, err, ctx.DeviceID)
			return
		}
		emails.EmailService.SendEmail(*account.Email, "Verify your AuthOne account", "authone_user_welcome", map[string]any{
			"OTP": otp,
		})
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *account.Email), "verify_account", time.Minute*10)
	} else {
		otp, err := auth.GenerateOTP(6, account.Phone.ISOCode)
		if err != nil {
			apperrors.FatalServerError(ctx, err, ctx.DeviceID)
			return
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", account.Phone.Prefix, account.Phone.LocalNumber), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx, err, ctx.DeviceID)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", account.Phone.ISOCode), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", account.Phone.ISOCode), "verify_account", time.Minute*10)
	}
	var savedDevice entities.Device
	for i, device := range account.Devices {
		if &device.ID == ctx.DeviceID {
			savedDevice = account.Devices[i]
			account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
			break
		}
	}
	account.Devices = append(account.Devices, entities.Device{
		ID:       savedDevice.ID,
		Name:     savedDevice.Name,
		Secret:   savedDevice.Secret,
		Verified: true,
	})
	_, err = userRepo.UpdatePartialByID(account.ID, map[string]any{
		"image":   fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"),
		"devices": account.Devices,
	})

	if err != nil {
		logger.Error("something went wrong when updating image status", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "image verified", nil, nil, nil, ctx.DeviceID)
}
