package controller

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	polymercore "authone.usepolymer.co/application/services/polymer-core"
	auth_usecases "authone.usepolymer.co/application/usecases/auth"
	user_usecases "authone.usepolymer.co/application/usecases/user"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"authone.usepolymer.co/infrastructure/messaging/emails"
	sms "authone.usepolymer.co/infrastructure/messaging/sms"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func KeyExchange(ctx *interfaces.ApplicationContext[dto.KeyExchangeDTO]) {
	// serverPublicKey, _ := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.Body.ClientPublicKey, ctx.DeviceID)
	// if serverPublicKey == nil {
	// 	return
	// }
	// server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusCreated, "key exchanged", hex.EncodeToString(serverPublicKey), nil, nil)
}

func AuthenticateUser(ctx *interfaces.ApplicationContext[dto.CreateUserDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	serverPublicKey, encryptedSecret := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.Body.ClientPublicKey, ctx.DeviceID)
	token, url, code, err := user_usecases.CreateUserUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent, encryptedSecret, ctx.DeviceName)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "authentication complete", map[string]any{
		"serverPublicKey": hex.EncodeToString(serverPublicKey),
		"url":             url,
		"code":            code,
		"token":           token,
	}, nil, nil, ctx.DeviceID)
}

func VerifyUserAccount(ctx *interfaces.ApplicationContext[any]) {
	userRepo := repository.UserRepo()
	filter := map[string]any{}
	if ctx.GetStringContextData("OTPEmail") != "" {
		filter["email"] = ctx.GetStringContextData("OTPEmail")
	} else {
		filter["phone.localNumber"] = ctx.GetBoolContextData("OTPPhone")
	}
	profile, err := userRepo.FindOneByFilter(filter, options.FindOne().SetProjection(map[string]any{
		"_id":       1,
		"firstName": 1,
		"lastName":  1,
		"deviceID":  1,
	}))
	if err != nil {
		logger.Error("an error occured while fetching user profile for verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	if profile == nil {
		apperrors.NotFoundError(ctx.Ctx, "Account not found", ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}

	token, err := auth.GenerateAuthToken(auth.ClaimsData{
		Email:     utils.GetStringPointer(ctx.GetStringContextData("OTPEmail")),
		UserID:    profile.ID,
		UserAgent: profile.UserAgent,
		// DeviceID:  profile.DeviceID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	if err != nil {
		logger.Error("an error occured while generating auth token after org verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	cache.Cache.CreateEntry(ctx.GetStringContextData("OTPToken"), true, time.Minute*5)
	hashedToken, _ := cryptography.CryptoHahser.HashString(*token, nil)
	cache.Cache.CreateEntry(profile.ID, hashedToken, time.Minute*time.Duration(10))
	polymercore.PolymerService.VerifyAccount(ctx.GetStringContextData("OTPEmail"))
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "email verified", token, nil, nil, ctx.DeviceID)
}

func VerifyOTP(ctx *interfaces.ApplicationContext[dto.VerifyOTPDTO]) {
	if ctx.Body.Phone == nil && ctx.Body.Email == nil {
		apperrors.ClientError(ctx.Ctx, "pass in either a phone number or email", nil, nil, ctx.DeviceID)
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
				apperrors.NotFoundError(ctx.Ctx, "otp has expired", ctx.DeviceID)
				return
			}
			d, err := cryptography.DecryptData(*otpRef, nil)
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
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "otp verified", token, nil, nil, ctx.DeviceID)
}

func ResendOTP(ctx *interfaces.ApplicationContext[dto.ResendOTPDTO]) {
	if ctx.Body.Email != nil {
		otp, err := auth.GenerateOTP(6, *ctx.Body.Email)
		if err != nil {
			apperrors.FatalServerError(ctx, err, ctx.DeviceID)
			return
		}
		emails.EmailService.SendEmail(*ctx.Body.Email, "AuthOne OTP", "authone_user_welcome", map[string]any{
			"OTP": otp,
		})
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *ctx.Body.Email), "verify_account", time.Minute*10)
	}
	if ctx.Body.Phone != nil {
		otp, err := auth.GenerateOTP(6, *ctx.Body.Phone)
		if err != nil {
			apperrors.FatalServerError(ctx, err, ctx.DeviceID)
			return
		}
		ref := sms.SMSService.SendOTP(fmt.Sprintf("%s%s", *ctx.Body.PhonePrefix, *ctx.Body.Phone), false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx, err, ctx.DeviceID)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", *ctx.Body.Phone), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *ctx.Body.Phone), "verify_account", time.Minute*10)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "otp sent", nil, nil, nil, ctx.DeviceID)
}
