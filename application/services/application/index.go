package services

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/constants"
	"gateman.io/application/repository"
	"gateman.io/application/utils"
	"gateman.io/entities"
	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/ipresolver"
	"gateman.io/infrastructure/ipresolver/types"
	"gateman.io/infrastructure/logger"
)

func ProcessUserSignUp(app *entities.Application, user *entities.User, ip string) (bool, string, map[string]any, map[string]any) {
	var eligible = true
	outstandingIDs := []string{}
	if app.Verifications != nil {
		for _, id := range *app.Verifications {
			if id.Name == "nin" {
				if user.NIN == nil && id.Required {
					outstandingIDs = append(outstandingIDs, "nin")
					eligible = false
				}
			} else {
				if user.BVN == nil && id.Required {
					outstandingIDs = append(outstandingIDs, "bvn")
					eligible = false
				}
			}
		}
	}

	loginLocaleRequested := false
	var results []string
	requestedFields := map[string]any{}
	userValue := reflect.ValueOf(*user)

	for _, field := range app.RequestedFields {
		if field.Name == "LoginLocale" {
			loginLocaleRequested = true
			continue
		}
		userField := userValue.FieldByName(field.Name)
		if !userField.IsValid() {
			results = append(results, field.Name)
			eligible = false
			continue
		}
		var userFieldData entities.KYCData[any]
		actualValue := userField.Interface()
		jsonBytes, _ := json.Marshal(actualValue)
		json.Unmarshal(jsonBytes, &userFieldData)

		// If Verified field doesn't exist or is not true, add to results
		if userFieldData.Value == nil || !userFieldData.Verified {
			results = append(results, field.Name)
			eligible = false

			// Edge case: If no ID is indicated for verification, default to "nin"
			if field.Verified && len(outstandingIDs) == 0 {
				outstandingIDs = append(outstandingIDs, "nin")
				eligible = false
			}
		}
		requestedFields[field.Name] = userFieldData.Value
	}
	var loginLocale *types.IPResult
	if loginLocaleRequested {
		result, err := ipresolver.IPResolverInstance.LookUp(ip)
		if err != nil {
			logger.Error("an error occured trying to get login locale for app signup", logger.LoggerOptions{
				Key: "err", Data: err,
			})
		}
		loginLocale = result
	}

	payload := map[string]any{}
	var msg string
	if eligible {
		msg = "Authentication successful"
		if loginLocale != nil {
			payload["loginLocale"] = loginLocale
		}
	} else {
		msg = "Additional info is required to sign up to this app"
		payload["missingIDs"] = outstandingIDs
		payload["unverifiedFields"] = results
	}
	return eligible, msg, payload, requestedFields
}

func GenerateAuthTokens(payload map[string]any, app *entities.Application, userAgent string, deviceID string, userID string, requestedFields map[string]any) (*map[string]any, error) {
	decryptedAppSigningKey, err := cryptography.DecryptData(app.AppSigningKey, nil)
	if err != nil {
		logger.Error("an error occured while generating auth token for app user sign in", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "app",
			Data: app,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		}, logger.LoggerOptions{
			Key:  "userAgent",
			Data: userAgent,
		}, logger.LoggerOptions{
			Key:  "deviceID",
			Data: deviceID,
		})
		return nil, err
	}
	requestedFieldsBytes, err := json.Marshal(requestedFields)
	if err != nil {
		logger.Error("an error occured while marshaling requested fields to []byte for app user sign in", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "app",
			Data: app,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		}, logger.LoggerOptions{
			Key:  "userAgent",
			Data: userAgent,
		}, logger.LoggerOptions{
			Key:  "deviceID",
			Data: deviceID,
		}, logger.LoggerOptions{
			Key:  "requestedFields",
			Data: requestedFields,
		})
		return nil, err
	}
	encrypted, err := cryptography.EncryptData(requestedFieldsBytes, utils.GetStringPointer(string(decryptedAppSigningKey)))
	if err != nil {
		logger.Error("an error occured while encrypting requested fields []byte for app user sign in", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "app",
			Data: app,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		}, logger.LoggerOptions{
			Key:  "userAgent",
			Data: userAgent,
		}, logger.LoggerOptions{
			Key:  "deviceID",
			Data: deviceID,
		}, logger.LoggerOptions{
			Key:  "requestedFields",
			Data: requestedFields,
		})
		return nil, err
	}
	var accessTokenTTL uint16
	var refreshTokenTTL uint32
	if os.Getenv("APP_ENV") == "production" {
		accessTokenTTL = app.AccessTokenTTL
		refreshTokenTTL = app.RefreshTokenTTL
	} else {
		accessTokenTTL = app.SandboxAccessTokenTTL
		refreshTokenTTL = app.SandboxRefreshTokenTTL
	}
	accessToken, err := auth.GenerateAppUserToken(auth.ClaimsData{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: int64(accessTokenTTL),
		UserAgent: userAgent,
		DeviceID:  deviceID,
		UserID:    userID,
	}, string(decryptedAppSigningKey), strings.ToLower(app.Name))
	if err != nil {
		logger.Error("an error occured while generating access token for app user sign in", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "app",
			Data: app,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		}, logger.LoggerOptions{
			Key:  "userAgent",
			Data: userAgent,
		}, logger.LoggerOptions{
			Key:  "deviceID",
			Data: deviceID,
		})
		return nil, err
	}
	refreshToken, err := auth.GenerateAppUserToken(auth.ClaimsData{
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: int64(refreshTokenTTL),
		UserAgent: userAgent,
		DeviceID:  deviceID,
		UserID:    userID,
	}, string(decryptedAppSigningKey), strings.ToLower(app.Name))
	if err != nil {
		logger.Error("an error occured while generating access token for app user sign in", logger.LoggerOptions{
			Key: "err", Data: err,
		}, logger.LoggerOptions{
			Key:  "app",
			Data: app,
		}, logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		}, logger.LoggerOptions{
			Key:  "userAgent",
			Data: userAgent,
		}, logger.LoggerOptions{
			Key:  "deviceID",
			Data: deviceID,
		})
	}
	payload["encryptedData"] = encrypted
	payload["refreshToken"] = refreshToken
	payload["accessToken"] = accessToken
	return &payload, nil
}

func CheckMonthlyLimit(ctx any, appID string, userID string, deviceID string) (block bool, err error) {
	activeSubRepo := repository.ActiveSubscriptionRepo()
	appActiveSub, err := activeSubRepo.FindOneByFilter(map[string]interface{}{
		"appID": appID,
	})
	if err != nil {
		logger.Error("an error occured trying to fetch apps active subscription", logger.LoggerOptions{
			Key:  "appID",
			Data: appID,
		}, logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		apperrors.UnknownError(ctx, err, nil, deviceID)
		return true, err
	}
	yearMonth := time.Now().Format("2006-01")

	appMAU := cache.Cache.CountSetMembers(fmt.Sprintf("application:%s:%s:mau", appID, yearMonth))
	if appActiveSub == nil || appActiveSub.ActiveSubName == entities.Free {
		if *appMAU >= constants.FREE_TIER_MAU_LIMIT {
			userTracked := cache.Cache.DoesItemExistInSet(fmt.Sprintf("application:%s:%s:mau", appID, yearMonth), userID)
			if !userTracked {
				apperrors.CustomError(ctx, "This app has hit it's free tier limit and cannot onboard new users", &constants.FREE_TIER_ACCOUNT_LIMIT_HIT, deviceID)
				return true, nil
			}
		}
		cache.Cache.CreateInSet(fmt.Sprintf("application:%s:%s:mau", appID, yearMonth), userID, time.Hour*24*365)
	} else {
		userTracked := cache.Cache.DoesItemExistInSet(fmt.Sprintf("application:%s:%s:mau", appID, yearMonth), userID)
		if !userTracked {
			cache.Cache.CreateInSet(fmt.Sprintf("application:%s:%s:mau", appID, yearMonth), userID, time.Hour*24*365)
			if *appMAU >= constants.PAID_TIER_FREE_MAU_LIMIT {
				if appActiveSub.ActiveSubName == entities.Essential {
					logger.Info("over limit user recorded on essential plan", logger.LoggerOptions{
						Key: "userID", Data: userID,
					})
					cache.Cache.IncrementField(fmt.Sprintf("application:%s:%s:essential-charge", appID, yearMonth), constants.ESSENTIAL_TIER_MAU_PRICE)
				} else {
					logger.Info("over limit user recorded on premium plan", logger.LoggerOptions{
						Key: "userID", Data: userID,
					})
					cache.Cache.IncrementField(fmt.Sprintf("application:%s:%s:premium-charge", appID, yearMonth), constants.PREMIUM_TIER_MAU_PRICE)
				}
			}
		}
	}

	return false, nil
}
