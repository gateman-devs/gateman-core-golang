package controller

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	auth_usecases "authone.usepolymer.co/application/usecases/auth"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/auth"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	sms "authone.usepolymer.co/infrastructure/messaging/sms"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func KeyExchange(ctx *interfaces.ApplicationContext[dto.KeyExchangeDTO]) {
	serverPublicKey := auth_usecases.InitiateKeyExchange(ctx.Ctx, ctx.DeviceID, ctx.Body.ClientPublicKey, ctx.DeviceID)
	if serverPublicKey == nil {
		return
	}
	server_response.Responder.UnEncryptedRespond(ctx.Ctx, http.StatusCreated, "key exchanged", hex.EncodeToString(serverPublicKey), nil, nil)
}

func VerifyOrg(ctx *interfaces.ApplicationContext[any]) {
	orgMemberRepo := repository.OrgMemberRepo()
	success, err := orgMemberRepo.UpdatePartialByFilter(map[string]any{
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
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	profile, err := orgMemberRepo.FindOneByFilter(map[string]any{
		"email": ctx.GetStringContextData("OTPEmail"),
	}, options.FindOne().SetProjection(map[string]any{
		"_id":       1,
		"firstName": 1,
		"lastName":  1,
		"deviceID":  1,
		"orgID":     1,
	}))
	if err != nil {
		logger.Error("an error occured while fetching user profile after org verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	token, err := auth.GenerateAuthToken(auth.ClaimsData{
		Email:     utils.GetStringPointer(ctx.GetStringContextData("OTPEmail")),
		FirstName: profile.FirstName,
		LastName:  profile.LastName,
		UserID:    profile.ID,
		UserAgent: profile.UserAgent,
		DeviceID:  profile.DeviceID,
		OrgID:     &profile.OrgID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Local().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	if err != nil {
		logger.Error("an error occured while generating auth token after org verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	cache.Cache.CreateEntry(ctx.GetStringContextData("OTPToken"), true, time.Minute*5)
	hashedToken, _ := cryptography.CryptoHahser.HashString(*token)
	cache.Cache.CreateEntry(profile.ID, hashedToken, time.Minute*time.Duration(10))
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
		OTPIntent: *otpIntent,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Local().Add(time.Minute * time.Duration(10)).Unix(), //lasts for 10 mins
	})
	if err != nil {
		apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "otp verified", token, nil, nil, ctx.DeviceID)
}

func LoginOrgMember(ctx *interfaces.ApplicationContext[dto.LoginDTO]) {
	attemptsMade := cache.Cache.FindOne(fmt.Sprintf("%s-login-attempts", ctx.Body.Email))
	var attempts int
	if attemptsMade != nil {
		attempts, _ = strconv.Atoi(*attemptsMade)
		if attempts == constants.MAX_LOGIN_ATTEMPTS {
			apperrors.AuthenticationError(ctx.Ctx, "You have exceeded the number of tries for this account and login has been disabled for the next 5 days to protect this account.", ctx.DeviceID)
			return
		}
	}
	orgMemberRepo := repository.OrgMemberRepo()
	account, err := orgMemberRepo.FindOneByFilter(map[string]interface{}{
		"email": ctx.Body.Email,
	})
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	if account == nil {
		apperrors.NotFoundError(ctx.Ctx, "Account not found or password mismatch", ctx.DeviceID)
		return
	}
	if !account.VerifiedEmail {
		apperrors.ClientError(ctx.Ctx, "Verify your email before attempting to login", nil, &constants.UNVERIFIED_EMAIL_LOGIN_ATTEMPT, ctx.DeviceID)
		return
	}
	match := cryptography.CryptoHahser.VerifyHashData(account.Password, ctx.Body.Password)
	if !match {
		attempts += 1
		cache.Cache.CreateEntry(fmt.Sprintf("%s-login-attempts", ctx.Body.Email), attempts, time.Hour*24*5) //lives for 5 days
		apperrors.ClientError(ctx.Ctx, fmt.Sprintf("Incorrect password. You have %d attempts left before this account is temporarily blocked", constants.MAX_LOGIN_ATTEMPTS-attempts), nil, nil, ctx.DeviceID)
		return
	}
	token, err := auth.GenerateAuthToken(auth.ClaimsData{
		Email:     utils.GetStringPointer(account.Email),
		FirstName: account.FirstName,
		LastName:  account.LastName,
		UserID:    account.ID,
		UserAgent: account.UserAgent,
		DeviceID:  account.DeviceID,
		OrgID:     &account.OrgID,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Local().Add(time.Minute * time.Duration(15)).Unix(), //lasts for 15 mins
	})
	if err != nil {
		logger.Error("an error occured while generating auth token after org verification", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return
	}
	hashedToken, _ := cryptography.CryptoHahser.HashString(*token)
	cache.Cache.CreateEntry(account.ID, hashedToken, time.Hour*1) // token should last for 10 mins
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "login successful", map[string]any{
		"account": account,
		"token":   token,
	}, nil, nil, ctx.DeviceID)
}
