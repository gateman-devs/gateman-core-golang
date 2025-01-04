package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
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
	identityverification "authone.usepolymer.co/infrastructure/identity_verification"
	identity_verification_types "authone.usepolymer.co/infrastructure/identity_verification/types"
	"authone.usepolymer.co/infrastructure/logger"
	sms "authone.usepolymer.co/infrastructure/messaging/sms"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetAccountImage(ctx *interfaces.ApplicationContext[any]) {
	exists, err := fileupload.FileUploader.CheckFileExists(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"))
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "azure", "500", err)
		return
	}
	if !exists {
		apperrors.ClientError(ctx.Ctx, "Image has not been uploaded. Request for a new url and upload image before attempting this request again.", nil, utils.GetUIntPointer(http.StatusBadRequest))
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	if !alive {
		apperrors.ClientError(ctx.Ctx, "Please make sure to take a clear picture of your face", nil, nil)
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	var savedDevice *entities.Device
	for i, device := range account.Devices {
		if device.ID == ctx.DeviceID {
			savedDevice = &account.Devices[i]
			account.Devices = append(account.Devices[:i], account.Devices[i+1:]...)
			break
		}
	}
	if savedDevice == nil {
		apperrors.NotFoundError(ctx.Ctx, "please onboard this device to continue registration")
		return
	}
	account.Devices = append(account.Devices, entities.Device{
		ID:        savedDevice.ID,
		Name:      savedDevice.Name,
		LastLogin: savedDevice.LastLogin,
		Verified:  true,
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
		apperrors.UnknownError(ctx.Ctx, err)
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
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 10 mins
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
		ExpiresAt:       time.Now().Add(time.Hour * 24 * 180).Unix(), //lasts for 10 mins
	})

	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*1)        // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 10 mins
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "image set", nil, nil, nil, accessToken, refreshToken)
}

func SetNINDetails(ctx *interfaces.ApplicationContext[dto.SetNINDetails]) {
	userRepo := repository.UserRepo()
	account, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"), options.FindOne().SetProjection(map[string]any{
		"nin": 1,
	}))
	if account.NIN != nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Seems you have verified your NIN already. You're good to go!", nil, nil, nil, nil, nil)
		return
	}
	hashedNIN, _ := cryptography.CryptoHahser.HashString(ctx.Body.NIN, []byte(os.Getenv("HASH_FIXED_SALT")))
	ninExists, _ := userRepo.CountDocs(map[string]interface{}{
		"nin": hashedNIN,
	})
	if ninExists != 0 {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "This NIN is already linked to another Gateman account.", nil, nil, nil, nil, nil)
		return
	}
	var nin identity_verification_types.NINData
	cachedNIN := cache.Cache.FindOne(string(hashedNIN))
	if cachedNIN == nil {
		fetchedNIN, _ := identityverification.IdentityVerifier.FetchNINDetails(ctx.Body.NIN)
		if fetchedNIN == nil {
			apperrors.NotFoundError(ctx.Ctx, "Invalid NIN provided")
			return
		}
		nin = *fetchedNIN
		ninByte, _ := nin.MarshalBinary()
		cache.Cache.CreateEntry(string(hashedNIN), ninByte, time.Hour*24*365) // save fetched nin details for a year
		return
	} else {
		err := json.Unmarshal([]byte(*cachedNIN), &nin)
		if err != nil {
			logger.Error("failed to marshal cached nin data", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{
				Key: "hashedNIN", Data: hashedNIN,
			})
			fetchedNIN, _ := identityverification.IdentityVerifier.FetchNINDetails(ctx.Body.NIN)
			if fetchedNIN == nil {
				apperrors.NotFoundError(ctx.Ctx, "Invalid NIN provided")
				return
			}
			ninByte, _ := nin.MarshalBinary()
			cache.Cache.CreateEntry(string(hashedNIN), ninByte, time.Hour*24*365) // save fetched nin details for a year
			return
		}
	}
	if os.Getenv("ENV") != "production" {
		nin.PhoneNumber = utils.GetStringPointer("00000000000")
	} else {
		nin.PhoneNumber = utils.GetStringPointer(fmt.Sprintf("234%s", *nin.PhoneNumber))
	}
	if ctx.GetStringContextData("Phone") != "" && nin.PhoneNumber != nil {
		if *nin.PhoneNumber == ctx.GetStringContextData("Phone") {
			parsedNINDOB, err := time.Parse("2006-01-02", "1990-01-01")
			if err != nil {
				logger.Error("failed to parse NIN DOB", logger.LoggerOptions{
					Key: "userID", Data: ctx.GetStringContextData("UserID"),
				}, logger.LoggerOptions{Key: "hashedNIN", Data: hashedNIN}, logger.LoggerOptions{
					Key: `err`, Data: err,
				})
				return
			}
			userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), map[string]any{
				"address": entities.Address{
					Value: &nin.Address,
				},
				"firstName": entities.KYCData[string]{
					Value:    &nin.FirstName,
					Verified: true,
				},
				"middleName": entities.KYCData[string]{
					Value:    nin.MiddleName,
					Verified: true,
				},
				"lastName": entities.KYCData[string]{
					Value:    &nin.LastName,
					Verified: true,
				},
				"gender": entities.KYCData[string]{
					Value:    &nin.Gender,
					Verified: true,
				},
				"dob": entities.KYCData[time.Time]{
					Value:    &parsedNINDOB,
					Verified: true,
				},
				"nin": hashedNIN,
			})
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "NIN Added", nil, nil, nil, nil, nil)
			return
		}
	}
	cache.Cache.CreateEntry(fmt.Sprintf("%s-nin", ctx.GetStringContextData("UserID")), hashedNIN, time.Hour*24*365)
	if nin.PhoneNumber != nil && *nin.PhoneNumber != "" {
		otp, err := auth.GenerateOTP(6, *nin.PhoneNumber)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err)
			return
		}
		ref := sms.SMSService.SendOTP(*nin.PhoneNumber, false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", *nin.PhoneNumber), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *nin.PhoneNumber), "verify_nin", time.Minute*10)
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, fmt.Sprintf("Verify OTP sent to ******%s", (*nin.PhoneNumber)[len(*nin.PhoneNumber)-4:]), nil, nil, nil, nil, nil)
	} else {
		logger.Error("Phone number not attached to NIN provided", logger.LoggerOptions{
			Key: "nin", Data: ctx.Body.NIN,
		}, logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.CustomError(ctx.Ctx, "Phone number not attached to NIN provided. Please reach out to support to resolve this issue")
	}
}

func VerifyNINDetails(ctx *interfaces.ApplicationContext[any]) {
	cachedNINNumber := cache.Cache.FindOne(fmt.Sprintf("%s-nin", ctx.GetStringContextData("UserID")))
	if cachedNINNumber == nil {
		logger.Error("cached nin number not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "NIN verification failed. Please restart verification process")
		return
	}
	cachedNIN := cache.Cache.FindOne(*cachedNINNumber)
	if cachedNIN == nil {
		logger.Error("cached nin not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "NIN verification failed. Please restart verification process")
		return
	}
	var nin identity_verification_types.NINData
	err := json.Unmarshal([]byte(*cachedNIN), &nin)
	if err != nil {
		logger.Error("failed to marshal cached nin data", logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		}, logger.LoggerOptions{
			Key: "cachedNIN", Data: cachedNIN,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	parsedNINDOB, err := time.Parse("2006-01-02", "1990-01-01")
	if err != nil {
		logger.Error("failed to parse NIN DOB", logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		}, logger.LoggerOptions{Key: "cachedNINNumber", Data: cachedNINNumber}, logger.LoggerOptions{
			Key: `err`, Data: err,
		})
		return
	}
	userRepo := repository.UserRepo()
	userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), map[string]any{
		"address": entities.Address{
			Value: &nin.Address,
		},
		"firstName": entities.KYCData[string]{
			Value:    &nin.FirstName,
			Verified: true,
		},
		"middleName": entities.KYCData[string]{
			Value:    nin.MiddleName,
			Verified: true,
		},
		"lastName": entities.KYCData[string]{
			Value:    &nin.LastName,
			Verified: true,
		},
		"gender": entities.KYCData[string]{
			Value:    &nin.Gender,
			Verified: true,
		},
		"dob": entities.KYCData[time.Time]{
			Value:    &parsedNINDOB,
			Verified: true,
		},
		"nin": cachedNINNumber,
	})
	cache.Cache.DeleteOne(fmt.Sprintf("%s-nin", ctx.GetStringContextData("UserID")))
	cache.Cache.DeleteOne(*cachedNINNumber)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "NIN Added", nil, nil, nil, nil, nil)
}
