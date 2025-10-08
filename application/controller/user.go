package controller

import (
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
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/biometric"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	identityverification "gateman.io/infrastructure/identity_verification"
	identity_verification_types "gateman.io/infrastructure/identity_verification/types"
	"gateman.io/infrastructure/logger"
	sms "gateman.io/infrastructure/messaging/sms"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SetAccountImage(ctx *interfaces.ApplicationContext[any]) {
	exists, err := fileupload.FileUploader.CheckFileExists(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"))
	if err != nil {
		apperrors.ExternalDependencyError(ctx.Ctx, "CLOUDFLARE", "500", err, ctx.DeviceID)
		return
	}
	if !exists {
		apperrors.ClientError(ctx.Ctx, "Image has not been uploaded. Request for a new url and upload image before attempting this request again.", nil, utils.GetUIntPointer(http.StatusBadRequest), ctx.DeviceID)
		return
	}
	// url, _ := fileupload.FileUploader.GeneratedSignedURL(fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage"), types.SignedURLPermission{
	// 	Read: true,
	// }, time.Minute*1)
	// alive, err := biometric.BiometricService.LivenessCheck(url)
	// if err != nil {
	// 	logger.Error("something went wrong when verifying image", logger.LoggerOptions{
	// 		Key:  "error",
	// 		Data: err,
	// 	})
	// 	apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
	// 	return
	// }
	// if !alive {
	// 	apperrors.ClientError(ctx.Ctx, "Please make sure to take a clear picture of your face", nil, nil, ctx.DeviceID)
	// 	return
	// }
	var availability_filter = map[string]any{}
	if ctx.GetStringContextData("Email") != "" {
		availability_filter["email"] = strings.ToLower(ctx.GetStringContextData("Email"))
	} else if ctx.GetStringContextData("Phone") != "" {
		availability_filter["phone.localNumber"] = ctx.GetStringContextData("Phone")
	}
	userRepo := repository.UserRepo()
	account, err := userRepo.FindOneByFilter(availability_filter)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
		apperrors.NotFoundError(ctx.Ctx, "please onboard this device to continue registration", &ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedRefreshToken, _ := cryptography.CryptoHahser.HashString(*refreshToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*1)        // token should last for 10 mins
	cache.Cache.CreateEntry(fmt.Sprintf("%s-refresh", string(hashedDeviceID)), hashedRefreshToken, time.Hour*24*180) // token should last for 10 mins
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "image set", map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}, nil, nil, &ctx.DeviceID)
}

func SetNINDetails(ctx *interfaces.ApplicationContext[dto.SetNINDetails]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	userRepo := repository.UserRepo()
	account, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"), options.FindOne().SetProjection(map[string]any{
		"nin": 1,
	}))
	if account.NIN != nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Seems you have verified your NIN already. You're good to go!", nil, nil, nil, &ctx.DeviceID)
		return
	}
	hashedNIN, _ := cryptography.CryptoHahser.HashString(ctx.Body.NIN, []byte(os.Getenv("HASH_FIXED_SALT")))
	ninExists, _ := userRepo.CountDocs(map[string]interface{}{
		"nin": hashedNIN,
	})
	if ninExists != 0 {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "This NIN is already linked to another Gateman account.", nil, nil, nil, &ctx.DeviceID)
		return
	}
	var nin identity_verification_types.NINData
	cachedNIN := cache.Cache.FindOne(string(hashedNIN))
	if cachedNIN == nil {
		fetchedNIN, _ := identityverification.IdentityVerifier.FetchNINDetails(ctx.Body.NIN)
		if fetchedNIN == nil {
			apperrors.NotFoundError(ctx.Ctx, "Invalid NIN provided", &ctx.DeviceID)
			return
		}
		nin = *fetchedNIN
		ninByte, _ := nin.MarshalBinary()
		cache.Cache.CreateEntry(string(hashedNIN), ninByte, time.Hour*24*365) // save fetched nin details for a year
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
				apperrors.NotFoundError(ctx.Ctx, "Invalid NIN provided", &ctx.DeviceID)
				return
			}
			ninByte, _ := nin.MarshalBinary()
			cache.Cache.CreateEntry(string(hashedNIN), ninByte, time.Hour*24*365) // save fetched nin details for a year
		}
	}
	if os.Getenv("APP_ENV") != "production" {
		nin.PhoneNumber = utils.GetStringPointer("00000000000")
	} else {
		nin.PhoneNumber = utils.GetStringPointer(fmt.Sprintf("234%s", *nin.PhoneNumber))
	}
	parsedNINDOB, err := time.Parse("2006-01-02", nin.DateOfBirth)
	if err != nil {
		logger.Error("failed to parse NIN DOB", logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		}, logger.LoggerOptions{Key: "hashedNIN", Data: hashedNIN}, logger.LoggerOptions{
			Key: `err`, Data: err,
		})
		return
	}
	if ctx.GetStringContextData("Phone") != "" && nin.PhoneNumber != nil {
		if *nin.PhoneNumber == ctx.GetStringContextData("Phone") || (nin.FirstName == *account.FirstName.Value && nin.LastName == *account.LastName.Value && parsedNINDOB.Equal(*account.DOB.Value)) {
			parsedNINDOB, err := time.Parse("2006-01-02", nin.DateOfBirth)
			if err != nil {
				logger.Error("failed to parse NIN DOB", logger.LoggerOptions{
					Key: "userID", Data: ctx.GetStringContextData("UserID"),
				}, logger.LoggerOptions{Key: "hashedNIN", Data: hashedNIN}, logger.LoggerOptions{
					Key: `err`, Data: err,
				})
				return
			}
			user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
			payload := map[string]any{"nin": hashedNIN}
			if user.Address == nil {
				payload["address"] = entities.Address{
					Value: &nin.Address,
				}
			}
			if user.FirstName == nil || !user.FirstName.Verified {
				payload["firstName"] = entities.KYCData[string]{
					Value:    &nin.FirstName,
					Verified: true,
				}
			}
			if user.MiddleName == nil || !user.MiddleName.Verified {
				payload["middleName"] = entities.KYCData[string]{
					Value:    nin.MiddleName,
					Verified: true,
				}
			}
			if user.LastName == nil || !user.LastName.Verified {
				payload["lastName"] = entities.KYCData[string]{
					Value:    &nin.LastName,
					Verified: true,
				}
			}
			if user.Gender == nil || !user.Gender.Verified {
				payload["gender"] = entities.KYCData[string]{
					Value:    &nin.Gender,
					Verified: true,
				}
			}
			if user.DOB == nil || !user.DOB.Verified {
				payload["dob"] = entities.KYCData[time.Time]{
					Value:    &parsedNINDOB,
					Verified: true,
				}

			}
			userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), payload)
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "NIN Added", nil, nil, nil, &ctx.DeviceID)
			return
		}
	}
	if nin.PhoneNumber != nil && *nin.PhoneNumber != "" {
		cache.Cache.CreateEntry(fmt.Sprintf("%s-nin", *nin.PhoneNumber), hashedNIN, time.Hour*24*365)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-nin-user", *nin.PhoneNumber), ctx.GetStringContextData("UserID"), time.Hour*24*365)
		otp, err := auth.GenerateOTP(6, *nin.PhoneNumber)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
			return
		}
		ref := sms.SMSService.SendOTP(*nin.PhoneNumber, false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", *nin.PhoneNumber), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", *nin.PhoneNumber), "verify_nin", time.Minute*10)
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, fmt.Sprintf("Verify OTP sent to ******%s", (*nin.PhoneNumber)[len(*nin.PhoneNumber)-4:]), nil, nil, nil, &ctx.DeviceID)
	} else {
		logger.Error("Phone number not attached to NIN provided", logger.LoggerOptions{
			Key: "nin", Data: ctx.Body.NIN,
		}, logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.CustomError(ctx.Ctx, "Phone number not attached to NIN provided. Please reach out to support to resolve this issue", nil, ctx.DeviceID)
	}
}

func VerifyNINDetails(ctx *interfaces.ApplicationContext[any]) {
	cachedNINNumber := cache.Cache.FindOne(fmt.Sprintf("%s-nin", ctx.GetStringContextData("OTPPhone")))
	if cachedNINNumber == nil {
		logger.Error("cached nin number not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "NIN verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	cachedNIN := cache.Cache.FindOne(*cachedNINNumber)
	if cachedNIN == nil {
		logger.Error("cached nin not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "NIN verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	userID := cache.Cache.FindOne(fmt.Sprintf("%s-nin-user", ctx.GetStringContextData("OTPPhone")))
	if userID == nil {
		logger.Error("userID nin not found", logger.LoggerOptions{
			Key:  "id",
			Data: userID,
		})
		apperrors.NotFoundError(ctx.Ctx, "NIN verification failed. Please restart verification process", &ctx.DeviceID)
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
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
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
	user, _ := userRepo.FindByID(*userID)
	payload := map[string]any{"nin": cachedNINNumber}
	if user.Address == nil {
		payload["address"] = entities.Address{
			Value: &nin.Address,
		}
	}
	if user.FirstName == nil || !user.FirstName.Verified {
		payload["firstName"] = entities.KYCData[string]{
			Value:    &nin.FirstName,
			Verified: true,
		}
	}
	if user.MiddleName == nil || !user.MiddleName.Verified {
		payload["middleName"] = entities.KYCData[string]{
			Value:    nin.MiddleName,
			Verified: true,
		}
	}
	if user.LastName == nil || !user.LastName.Verified {
		payload["lastName"] = entities.KYCData[string]{
			Value:    &nin.LastName,
			Verified: true,
		}
	}
	if user.Gender == nil || !user.Gender.Verified {
		payload["gender"] = entities.KYCData[string]{
			Value:    &nin.Gender,
			Verified: true,
		}
	}
	if user.DOB == nil || !user.DOB.Verified {
		payload["dob"] = entities.KYCData[time.Time]{
			Value:    &parsedNINDOB,
			Verified: true,
		}

	}
	userRepo.UpdatePartialByID(*userID, payload)
	cache.Cache.DeleteOne(fmt.Sprintf("%s-nin", ctx.GetStringContextData("UserID")))
	cache.Cache.DeleteOne(*cachedNINNumber)

	var phone *string
	if user.Phone != nil {
		phone = utils.GetStringPointer(fmt.Sprintf("%s%s", user.Phone.Prefix, user.Phone.LocalNumber))
	}
	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          user.ID,
		UserAgent:       user.UserAgent,
		Email:           user.Email,
		VerifiedAccount: user.VerifiedAccount,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "NIN Added", map[string]any{
		"accessToken": accessToken,
	}, nil, nil, &ctx.DeviceID)
}

func SetBVNDetails(ctx *interfaces.ApplicationContext[dto.SetBVNDetails]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	userRepo := repository.UserRepo()
	account, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"), options.FindOne().SetProjection(map[string]any{
		"bvn": 1,
	}))
	if account.BVN != nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Seems you have verified your BVN already. You're good to go!", nil, nil, nil, &ctx.DeviceID)
		return
	}
	hashedBVN, _ := cryptography.CryptoHahser.HashString(ctx.Body.BVN, []byte(os.Getenv("HASH_FIXED_SALT")))
	bvnExists, _ := userRepo.CountDocs(map[string]interface{}{
		"bvn": hashedBVN,
	})
	if bvnExists != 0 {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "This BVN is already linked to another Gateman account.", nil, nil, nil, &ctx.DeviceID)
		return
	}
	var bvn identity_verification_types.BVNData
	cachedBVN := cache.Cache.FindOne(string(hashedBVN))
	if cachedBVN == nil {
		fetchedBVN, _ := identityverification.IdentityVerifier.FetchBVNDetails(ctx.Body.BVN)
		if fetchedBVN == nil {
			apperrors.NotFoundError(ctx.Ctx, "Invalid BVN provided", &ctx.DeviceID)
			return
		}
		bvn = *fetchedBVN
		bvnByte, _ := bvn.MarshalBinary()
		cache.Cache.CreateEntry(string(hashedBVN), bvnByte, time.Hour*24*365) // save fetched bvn details for a year
	} else {
		err := json.Unmarshal([]byte(*cachedBVN), &bvn)
		if err != nil {
			logger.Error("failed to marshal cached bvn data", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{
				Key: "hashedBVN", Data: hashedBVN,
			})
			fetchedBVN, _ := identityverification.IdentityVerifier.FetchBVNDetails(ctx.Body.BVN)
			if fetchedBVN == nil {
				apperrors.NotFoundError(ctx.Ctx, "Invalid BVN provided", &ctx.DeviceID)
				return
			}
			bvnByte, _ := bvn.MarshalBinary()
			cache.Cache.CreateEntry(string(hashedBVN), bvnByte, time.Hour*24*365) // save fetched bvn details for a year
		}
	}
	if os.Getenv("APP_ENV") != "production" {
		bvn.PhoneNumber = "00000000000"
	} else {
		bvn.PhoneNumber = fmt.Sprintf("234%s", bvn.PhoneNumber)
	}
	if ctx.GetStringContextData("Phone") != "" && bvn.PhoneNumber != "" {
		parsedBVNDOB, err := time.Parse("2006-01-02", bvn.DateOfBirth)
		if err != nil {
			logger.Error("failed to parse BVN DOB", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{Key: "hashedBVN", Data: string(hashedBVN)}, logger.LoggerOptions{
				Key: `err`, Data: err,
			})
			return
		}
		if bvn.PhoneNumber == ctx.GetStringContextData("Phone") || (bvn.FirstName == *account.FirstName.Value && bvn.LastName == *account.LastName.Value && parsedBVNDOB.Equal(*account.DOB.Value)) {
			parsedBVNDOB, err := time.Parse("2006-01-02", bvn.DateOfBirth)
			if err != nil {
				logger.Error("failed to parse BVN DOB", logger.LoggerOptions{
					Key: "userID", Data: ctx.GetStringContextData("UserID"),
				}, logger.LoggerOptions{Key: "hashedBVN", Data: string(hashedBVN)}, logger.LoggerOptions{
					Key: `err`, Data: err,
				})
				return
			}

			user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
			payload := map[string]any{"bvn": string(hashedBVN)}
			if user.Address == nil {
				payload["address"] = entities.Address{
					Value: &bvn.Address,
				}
			}
			if user.FirstName == nil || !user.FirstName.Verified {
				payload["firstName"] = entities.KYCData[string]{
					Value:    &bvn.FirstName,
					Verified: true,
				}
			}
			if user.MiddleName == nil || !user.MiddleName.Verified {
				payload["middleName"] = entities.KYCData[string]{
					Value:    bvn.MiddleName,
					Verified: true,
				}
			}
			if user.LastName == nil || !user.LastName.Verified {
				payload["lastName"] = entities.KYCData[string]{
					Value:    &bvn.LastName,
					Verified: true,
				}
			}
			if user.Gender == nil || !user.Gender.Verified {
				payload["gender"] = entities.KYCData[string]{
					Value:    &bvn.Gender,
					Verified: true,
				}
			}
			if user.DOB == nil || !user.DOB.Verified {
				payload["dob"] = entities.KYCData[time.Time]{
					Value:    &parsedBVNDOB,
					Verified: true,
				}

			}
			userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), payload)
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "BVN Added", nil, nil, nil, &ctx.DeviceID)
			return
		}
	}
	if bvn.PhoneNumber != "" {
		cache.Cache.CreateEntry(fmt.Sprintf("%s-bvn", bvn.PhoneNumber), hashedBVN, time.Hour*24*365)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-bvn-user", bvn.PhoneNumber), ctx.GetStringContextData("UserID"), time.Hour*24*365)
		otp, err := auth.GenerateOTP(6, bvn.PhoneNumber)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
			return
		}
		ref := sms.SMSService.SendOTP(bvn.PhoneNumber, false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", bvn.PhoneNumber), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", bvn.PhoneNumber), "verify_bvn", time.Minute*10)
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, fmt.Sprintf("Verify OTP sent to ******%s", (bvn.PhoneNumber)[len(bvn.PhoneNumber)-4:]), nil, nil, nil, &ctx.DeviceID)
	} else {
		logger.Error("Phone number not attached to BVN provided", logger.LoggerOptions{
			Key: "bvn", Data: ctx.Body.BVN,
		}, logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.CustomError(ctx.Ctx, "Phone number not attached to BVN provided. Please reach out to support to resolve this issue", nil, ctx.DeviceID)
	}
}

func VerifyBVNDetails(ctx *interfaces.ApplicationContext[any]) {
	cachedBVNNumber := cache.Cache.FindOne(fmt.Sprintf("%s-bvn", ctx.GetStringContextData("OTPPhone")))
	if cachedBVNNumber == nil {
		logger.Error("cached BVN number not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "BVN verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	cachedBVN := cache.Cache.FindOne(*cachedBVNNumber)
	if cachedBVN == nil {
		logger.Error("cached BVN not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "BVN verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	userID := cache.Cache.FindOne(fmt.Sprintf("%s-bvn-user", ctx.GetStringContextData("OTPPhone")))
	if userID == nil {
		logger.Error("userID bvn not found", logger.LoggerOptions{
			Key:  "id",
			Data: userID,
		})
		apperrors.NotFoundError(ctx.Ctx, "BVN verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	var bvn identity_verification_types.BVNData
	err := json.Unmarshal([]byte(*cachedBVN), &bvn)
	if err != nil {
		logger.Error("failed to marshal cached bvn data", logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		}, logger.LoggerOptions{
			Key: "cachedBVN", Data: cachedBVN,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	parsedBVNDOB, err := time.Parse("02-Jan-2006", bvn.DateOfBirth)
	if err != nil {
		logger.Error("failed to parse BVN DOB", logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		}, logger.LoggerOptions{Key: "cachedBVNNumber", Data: cachedBVNNumber}, logger.LoggerOptions{
			Key: `err`, Data: err,
		})
		return
	}
	userRepo := repository.UserRepo()
	user, _ := userRepo.FindByID(*userID)
	payload := map[string]any{"bvn": cachedBVNNumber}
	if user.Address == nil {
		payload["address"] = entities.Address{
			Value: &bvn.Address,
		}
	}
	if user.FirstName == nil || !user.FirstName.Verified {
		payload["firstName"] = entities.KYCData[string]{
			Value:    &bvn.FirstName,
			Verified: true,
		}
	}
	if user.MiddleName == nil || !user.MiddleName.Verified {
		payload["middleName"] = entities.KYCData[string]{
			Value:    bvn.MiddleName,
			Verified: true,
		}
	}
	if user.LastName == nil || !user.LastName.Verified {
		payload["lastName"] = entities.KYCData[string]{
			Value:    &bvn.LastName,
			Verified: true,
		}
	}
	if user.Gender == nil || !user.Gender.Verified {
		payload["gender"] = entities.KYCData[string]{
			Value:    &bvn.Gender,
			Verified: true,
		}
	}
	if user.DOB == nil || !user.DOB.Verified {
		payload["dob"] = entities.KYCData[time.Time]{
			Value:    &parsedBVNDOB,
			Verified: true,
		}

	}
	userRepo.UpdatePartialByID(*userID, payload)
	cache.Cache.DeleteOne(fmt.Sprintf("%s-bvn", ctx.GetStringContextData("UserID")))
	cache.Cache.DeleteOne(*cachedBVNNumber)

	var phone *string
	if user.Phone != nil {
		phone = utils.GetStringPointer(fmt.Sprintf("%s%s", user.Phone.Prefix, user.Phone.LocalNumber))
	}
	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          user.ID,
		UserAgent:       user.UserAgent,
		Email:           user.Email,
		VerifiedAccount: user.VerifiedAccount,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "BVN Added", map[string]any{
		"accessToken": accessToken,
	}, nil, nil, &ctx.DeviceID)
}

func SetDriversLicenseDetails(ctx *interfaces.ApplicationContext[dto.SetDriversLicenseDetails]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	userRepo := repository.UserRepo()
	account, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"), options.FindOne().SetProjection(map[string]any{
		"driverID":  1,
		"image":     1,
		"firstName": 1,
		"lastName":  1,
		"dob":       1,
	}))
	if account.DriverID != nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Seems you have verified your Drivers License already. You're good to go!", nil, nil, nil, &ctx.DeviceID)
		return
	}
	hashedDriversID, _ := cryptography.CryptoHahser.HashString(ctx.Body.DriverID, []byte(os.Getenv("HASH_FIXED_SALT")))
	driversIDExists, _ := userRepo.CountDocs(map[string]interface{}{
		"driverID": hashedDriversID,
	})
	if driversIDExists != 0 {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "This Drivers License is already linked to another Gateman account.", nil, nil, nil, &ctx.DeviceID)
		return
	}
	var driverID identity_verification_types.DriversID
	cachedDriverID := cache.Cache.FindOne(string(hashedDriversID))
	if cachedDriverID == nil {
		fetchedDriverID, err := identityverification.IdentityVerifier.FetchDriverIDDetails(ctx.Body.DriverID)
		if fetchedDriverID == nil {
			apperrors.NotFoundError(ctx.Ctx, "Invalid Driver ID provided", &ctx.DeviceID)
			return
		}
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		driverID = *fetchedDriverID
		driverIDByte, _ := driverID.MarshalBinary()
		cache.Cache.CreateEntry(string(hashedDriversID), driverIDByte, time.Hour*24*365) // save fetched driver id details for a year
	} else {
		err := json.Unmarshal([]byte(*cachedDriverID), &driverID)
		if err != nil {
			logger.Error("failed to marshal cached Driver ID data", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{
				Key: "hashedDriverID", Data: hashedDriversID,
			})
			fetchedDriverID, _ := identityverification.IdentityVerifier.FetchDriverIDDetails(ctx.Body.DriverID)
			if fetchedDriverID == nil {
				apperrors.NotFoundError(ctx.Ctx, "Invalid Driver ID provided", &ctx.DeviceID)
				return
			}
			driverIDByte, _ := driverID.MarshalBinary()
			cache.Cache.CreateEntry(string(hashedDriversID), driverIDByte, time.Hour*24*365) // save fetched driver id details for a year
		}
	}
	accountImgURL, _ := fileupload.FileUploader.GeneratedSignedURL(account.Image, types.SignedURLPermission{
		Read: true,
	}, time.Minute*1)
	success, _ := biometric.BiometricService.CompareFaces(&driverID.Photo, accountImgURL)
	if !success.Success {
		parsedDriverIDDOB, err := time.Parse("02-01-2006", driverID.BirthDate)
		if err != nil {
			logger.Error("failed to parse Driver ID DOB", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{Key: "hashedDriverID", Data: string(hashedDriversID)}, logger.LoggerOptions{
				Key: `err`, Data: err,
			})
			return
		}
		if driverID.FirstName == *account.FirstName.Value && driverID.LastName == *account.LastName.Value && parsedDriverIDDOB.Equal(*account.DOB.Value) {
			success.Success = true
		}
	}
	if !success.Success {
		parsedDriverIDDOB, err := time.Parse("02-01-2006", driverID.BirthDate)
		if err != nil {
			logger.Error("failed to parse Driver ID DOB", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{Key: "hashedDriverID", Data: string(hashedDriversID)}, logger.LoggerOptions{
				Key: `err`, Data: err,
			})
			return
		}

		user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
		payload := map[string]any{"driverID": string(hashedDriversID)}
		if user.FirstName == nil || !user.FirstName.Verified {
			payload["firstName"] = entities.KYCData[string]{
				Value: &driverID.FirstName,
			}
		}
		if user.MiddleName == nil || !user.MiddleName.Verified {
			payload["middleName"] = entities.KYCData[string]{
				Value: driverID.MiddleName,
			}
		}
		if user.LastName == nil || !user.LastName.Verified {
			payload["lastName"] = entities.KYCData[string]{
				Value: &driverID.LastName,
			}
		}
		if user.Gender == nil || !user.Gender.Verified {
			payload["gender"] = entities.KYCData[string]{
				Value: &driverID.Gender,
			}
		}
		if user.DOB == nil || !user.DOB.Verified {
			payload["dob"] = entities.KYCData[time.Time]{
				Value: &parsedDriverIDDOB,
			}
		}
		userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), payload)
	} else {
		parsedDriverIDDOB, err := time.Parse("02-01-2006", driverID.BirthDate)
		if err != nil {
			logger.Error("failed to parse Driver ID DOB", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{Key: "hashedDriverID", Data: string(hashedDriversID)}, logger.LoggerOptions{
				Key: `err`, Data: err,
			})
			return
		}

		user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
		payload := map[string]any{"driverID": string(hashedDriversID)}
		if user.FirstName == nil || !user.FirstName.Verified {
			payload["firstName"] = entities.KYCData[string]{
				Value:    &driverID.FirstName,
				Verified: true,
			}
		}
		if user.MiddleName == nil || !user.MiddleName.Verified {
			payload["middleName"] = entities.KYCData[string]{
				Value:    driverID.MiddleName,
				Verified: true,
			}
		}
		if user.LastName == nil || !user.LastName.Verified {
			payload["lastName"] = entities.KYCData[string]{
				Value:    &driverID.LastName,
				Verified: true,
			}
		}
		if user.Gender == nil || !user.Gender.Verified {
			payload["gender"] = entities.KYCData[string]{
				Value:    &driverID.Gender,
				Verified: true,
			}
		}
		if user.DOB == nil || !user.DOB.Verified {
			payload["dob"] = entities.KYCData[time.Time]{
				Value:    &parsedDriverIDDOB,
				Verified: true,
			}

		}
		userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), payload)
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "drivers license added", nil, nil, nil, &ctx.DeviceID)
}

func SetVoterIDDetails(ctx *interfaces.ApplicationContext[dto.SetVoterIDDetails]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	userRepo := repository.UserRepo()
	account, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"), options.FindOne().SetProjection(map[string]any{
		"voterID": 1,
	}))
	if account.VoterID != nil {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Seems you have verified your Voter ID already. You're good to go!", nil, nil, nil, &ctx.DeviceID)
		return
	}
	hashedVoterID, _ := cryptography.CryptoHahser.HashString(ctx.Body.VoterID, []byte(os.Getenv("HASH_FIXED_SALT")))
	voterIDExists, _ := userRepo.CountDocs(map[string]interface{}{
		"voterID": hashedVoterID,
	})
	if voterIDExists != 0 {
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "This Voter ID is already linked to another Gateman account.", nil, nil, nil, &ctx.DeviceID)
		return
	}
	var voterID identity_verification_types.VoterID
	cachedVoterID := cache.Cache.FindOne(string(hashedVoterID))
	if cachedVoterID == nil {
		fetchedVoterID, _ := identityverification.IdentityVerifier.FetchVoterIDDetails(ctx.Body.VoterID)
		if fetchedVoterID == nil {
			apperrors.NotFoundError(ctx.Ctx, "Invalid Voter ID provided", &ctx.DeviceID)
			return
		}
		voterID = *fetchedVoterID
		bvnByte, _ := voterID.MarshalBinary()
		cache.Cache.CreateEntry(string(hashedVoterID), bvnByte, time.Hour*24*365) // save fetched bvn details for a year
	} else {
		err := json.Unmarshal([]byte(*cachedVoterID), &voterID)
		if err != nil {
			logger.Error("failed to marshal cached Voter ID data", logger.LoggerOptions{
				Key: "userID", Data: ctx.GetStringContextData("UserID"),
			}, logger.LoggerOptions{
				Key: "hashedBVN", Data: hashedVoterID,
			})
			fetchedBVN, _ := identityverification.IdentityVerifier.FetchVoterIDDetails(ctx.Body.VoterID)
			if fetchedBVN == nil {
				apperrors.NotFoundError(ctx.Ctx, "Invalid Voter ID provided", &ctx.DeviceID)
				return
			}
			bvnByte, _ := voterID.MarshalBinary()
			cache.Cache.CreateEntry(string(hashedVoterID), bvnByte, time.Hour*24*365) // save fetched bvn details for a year
		}
	}
	if os.Getenv("APP_ENV") != "production" {
		voterID.Phone = "00000000000"
	} else {
		voterID.Phone = fmt.Sprintf("234%s", voterID.Phone)
	}
	if ctx.GetStringContextData("Phone") != "" && voterID.Phone != "" {
		// parsedBVNDOB, err := time.Parse("2006-01-02", voterID.DateOfBirth)
		// if err != nil {
		// 	logger.Error("failed to parse Voter ID DOB", logger.LoggerOptions{
		// 		Key: "userID", Data: ctx.GetStringContextData("UserID"),
		// 	}, logger.LoggerOptions{Key: "hashedBVN", Data: string(hashedVoterID)}, logger.LoggerOptions{
		// 		Key: `err`, Data: err,
		// 	})
		// 	return
		// }
		names := strings.Split(voterID.FullName, " ")
		if voterID.Phone == ctx.GetStringContextData("Phone") || (names[2] == *account.FirstName.Value && names[0] == *account.LastName.Value) {
			// parsedDOB, err := time.Parse("2006-01-02", voterID.DateOfBirth)
			// if err != nil {
			// 	logger.Error("failed to parse Voter ID DOB", logger.LoggerOptions{
			// 		Key: "userID", Data: ctx.GetStringContextData("UserID"),
			// 	}, logger.LoggerOptions{Key: "hashedBVN", Data: string(hashedVoterID)}, logger.LoggerOptions{
			// 		Key: `err`, Data: err,
			// 	})
			// 	return
			// }

			user, _ := userRepo.FindByID(ctx.GetStringContextData("UserID"))
			payload := map[string]any{"voterID": string(hashedVoterID)}
			if user.Address == nil {
				payload["address"] = entities.Address{
					Value: &voterID.Address,
				}
			}
			if user.FirstName == nil || !user.FirstName.Verified {
				payload["firstName"] = entities.KYCData[string]{
					Value:    &names[0],
					Verified: true,
				}
			}
			if user.MiddleName == nil || !user.MiddleName.Verified {
				payload["middleName"] = entities.KYCData[string]{
					Value:    &names[1],
					Verified: true,
				}
			}
			if user.LastName == nil || !user.LastName.Verified {
				payload["lastName"] = entities.KYCData[string]{
					Value:    &names[2],
					Verified: true,
				}
			}
			if user.Gender == nil || !user.Gender.Verified {
				payload["gender"] = entities.KYCData[string]{
					Value:    &voterID.Gender,
					Verified: true,
				}
			}
			// if user.DOB == nil || !user.DOB.Verified {
			// 	payload["dob"] = entities.KYCData[time.Time]{
			// 		Value:    &parsedDOB,
			// 		Verified: true,
			// 	}

			// }
			userRepo.UpdatePartialByID(ctx.GetStringContextData("UserID"), payload)
			server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Voter ID Added", nil, nil, nil, &ctx.DeviceID)
			return
		}
	}
	if voterID.Phone != "" {
		cache.Cache.CreateEntry(fmt.Sprintf("%s-voter-id", voterID.Phone), hashedVoterID, time.Hour*24*365)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-voter-id-user", voterID.Phone), ctx.GetStringContextData("UserID"), time.Hour*24*365)
		otp, err := auth.GenerateOTP(6, voterID.Phone)
		if err != nil {
			apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
			return
		}
		ref := sms.SMSService.SendOTP(voterID.Phone, false, otp)
		encryptedRef, err := cryptography.EncryptData([]byte(*ref), nil)
		if err != nil {
			apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
			return
		}
		cache.Cache.CreateEntry(fmt.Sprintf("%s-sms-otp-ref", voterID.Phone), *encryptedRef, time.Minute*10)
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", voterID.Phone), "verify_voter_id", time.Minute*10)
		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, fmt.Sprintf("Verify OTP sent to ******%s", (voterID.Phone)[len(voterID.Phone)-4:]), nil, nil, nil, &ctx.DeviceID)
	} else {
		logger.Error("Phone number not attached to Voter ID provided", logger.LoggerOptions{
			Key: "Voter ID", Data: ctx.Body.VoterID,
		}, logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.CustomError(ctx.Ctx, "Phone number not attached to Voter ID provided. Please reach out to support to resolve this issue", nil, ctx.DeviceID)
	}
}

func VerifyVoterIDDetails(ctx *interfaces.ApplicationContext[any]) {
	cachedVoterIDNumber := cache.Cache.FindOne(fmt.Sprintf("%s-voter-id", ctx.GetStringContextData("OTPPhone")))
	if cachedVoterIDNumber == nil {
		logger.Error("cached Voter ID number not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "Voter ID verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	cachedVoterID := cache.Cache.FindOne(*cachedVoterIDNumber)
	if cachedVoterID == nil {
		logger.Error("cached Voter ID not found", logger.LoggerOptions{
			Key:  "id",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.NotFoundError(ctx.Ctx, "Voter ID verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	userID := cache.Cache.FindOne(fmt.Sprintf("%s-voter-id-user", ctx.GetStringContextData("OTPPhone")))
	if userID == nil {
		logger.Error("userID Voter ID not found", logger.LoggerOptions{
			Key:  "id",
			Data: userID,
		})
		apperrors.NotFoundError(ctx.Ctx, "Voter ID verification failed. Please restart verification process", &ctx.DeviceID)
		return
	}
	var voterID identity_verification_types.VoterID
	err := json.Unmarshal([]byte(*cachedVoterID), &voterID)
	if err != nil {
		logger.Error("failed to marshal cached Voter ID data", logger.LoggerOptions{
			Key: "userID", Data: ctx.GetStringContextData("UserID"),
		}, logger.LoggerOptions{
			Key: "cachedVoterID", Data: cachedVoterID,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	// parsedDOB, err := time.Parse("2006-01-02", voterID.DateOfBirth)
	// if err != nil {
	// 	logger.Error("failed to parse Voter ID DOB", logger.LoggerOptions{
	// 		Key: "userID", Data: ctx.GetStringContextData("UserID"),
	// 	}, logger.LoggerOptions{Key: "cachedBVoterID", Data: cachedVoterID}, logger.LoggerOptions{
	// 		Key: `err`, Data: err,
	// 	})
	// 	return
	// }
	userRepo := repository.UserRepo()
	user, _ := userRepo.FindByID(*userID)
	payload := map[string]any{"voterID": cachedVoterIDNumber}
	if user.Address == nil {
		payload["address"] = entities.Address{
			Value: &voterID.Address,
		}
	}
	names := strings.Split(voterID.FullName, " ")
	if user.FirstName == nil || !user.FirstName.Verified {
		payload["firstName"] = entities.KYCData[string]{
			Value:    &names[0],
			Verified: true,
		}
	}
	if user.MiddleName == nil || !user.MiddleName.Verified {
		payload["middleName"] = entities.KYCData[string]{
			Value:    &names[1],
			Verified: true,
		}
	}
	if user.LastName == nil || !user.LastName.Verified {
		payload["lastName"] = entities.KYCData[string]{
			Value:    &names[2],
			Verified: true,
		}
	}
	if user.Gender == nil || !user.Gender.Verified {
		payload["gender"] = entities.KYCData[string]{
			Value:    &voterID.Gender,
			Verified: true,
		}
	}
	// if user.DOB == nil || !user.DOB.Verified {
	// 	payload["dob"] = entities.KYCData[time.Time]{
	// 		Value:    &parsedDOB,
	// 		Verified: true,
	// 	}
	// }
	userRepo.UpdatePartialByID(*userID, payload)
	cache.Cache.DeleteOne(fmt.Sprintf("%s-voter-id", ctx.GetStringContextData("UserID")))
	cache.Cache.DeleteOne(*cachedVoterID)

	var phone *string
	if user.Phone != nil {
		phone = utils.GetStringPointer(fmt.Sprintf("%s%s", user.Phone.Prefix, user.Phone.LocalNumber))
	}
	accessToken, err := auth.GenerateAuthToken(auth.ClaimsData{
		UserID:          user.ID,
		UserAgent:       user.UserAgent,
		Email:           user.Email,
		VerifiedAccount: user.VerifiedAccount,
		PhoneNum:        phone,
		DeviceID:        ctx.DeviceID,
		TokenType:       auth.AccessToken,
		IssuedAt:        time.Now().Unix(),
		ExpiresAt:       time.Now().Add(time.Hour * 1).Unix(), //lasts for 1 hr
	})
	hashedAccessToken, _ := cryptography.CryptoHahser.HashString(*accessToken, nil)
	hashedDeviceID, _ := cryptography.CryptoHahser.HashString(ctx.DeviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	cache.Cache.CreateEntry(fmt.Sprintf("%s-access", string(hashedDeviceID)), hashedAccessToken, time.Hour*24)
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "Voter ID Added", map[string]any{
		"accessToken": accessToken,
	}, nil, nil, &ctx.DeviceID)
}
