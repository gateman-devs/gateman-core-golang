package auth

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
)

const otpChars = "1234567890"

func GenerateOTP(length int, channel string) (*string, error) {
	var otp string
	if os.Getenv("ENV") == "staging" || os.Getenv("ENV") == "development" {
		otp = "000000"
	} else {
		buffer := make([]byte, length)
		_, err := rand.Read(buffer)
		if err != nil {
			return nil, err
		}
		otpCharsLength := len(otpChars)
		for i := 0; i < length; i++ {
			buffer[i] = otpChars[int(buffer[i])%otpCharsLength]
		}
		otp = string(buffer)
	}
	otpSaved := saveOTP(channel, otp)
	if !otpSaved {
		return nil, errors.New("could not save otp")
	}
	return &otp, nil
}

func saveOTP(channel string, otp string) bool {
	hashedOTP, err := cryptography.CryptoHahser.HashString(otp, nil)
	if err != nil {
		logger.Error("auth module error - error while saving otp", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return false
	}
	return cache.Cache.CreateEntry(fmt.Sprintf("%s-otp", channel), string(hashedOTP), 10*time.Minute) // otp is valid for 10 mins
}

func VerifyOTP(key string, otp string) (string, bool) {
	data := cache.Cache.FindOne(fmt.Sprintf("%s-otp", key))
	if data == nil {
		logger.Info(fmt.Sprintf("%s otp not found", key))
		return "this otp has expired", false
	}
	success := cryptography.CryptoHahser.VerifyHashData(*data, otp)
	if !success {
		return "wrong otp provided", false
	}
	cache.Cache.DeleteOne(fmt.Sprintf("%s-otp", key))
	return "", true
}

func GenerateAuthToken(claimsData ClaimsData) (*string, error) {
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":             os.Getenv("JWT_ISSUER"),
		"userID":          claimsData.UserID,
		"exp":             claimsData.ExpiresAt,
		"email":           claimsData.Email,
		"firstName":       claimsData.FirstName,
		"lastName":        claimsData.LastName,
		"iat":             claimsData.IssuedAt,
		"deviceID":        claimsData.DeviceID,
		"userAgent":       claimsData.UserAgent,
		"intent":          claimsData.Intent,
		"phone":           claimsData.PhoneNum,
		"verifiedAccount": claimsData.VerifiedAccount,
		"tokenType":       claimsData.TokenType,
	}).SignedString([]byte(os.Getenv("JWT_SIGNING_KEY")))
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func GenerateAppUserToken(payload any, signingKey string) (*string, error) {
	key := []byte(signingKey)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("could not serialise json for GenerateAppUserToken", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		return nil, err
	}
	encrypted, err := jwe.Encrypt(jsonData, jwe.WithKey(jwa.A256GCMKW, key))
	if err != nil {
		logger.Error("could not encrypt json for GenerateAppUserToken", logger.LoggerOptions{
			Key: "err", Data: err,
		})
		return nil, err
	}
	return utils.GetStringPointer(string(encrypted)), nil
}

func GenerateInterserviceAuthToken(claimsData InterserviceClaimsData) (*string, error) {
	tokenString, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"iss":    os.Getenv("INTERSERVICE_JWT_ISSUER"),
		"exp":    claimsData.ExpiresAt,
		"origin": claimsData.Origination,
		"iat":    claimsData.IssuedAt,
	}).SignedString([]byte(os.Getenv("INTERSERVICE_JWT_SIGNING_KEY")))
	if err != nil {
		return nil, err
	}
	return &tokenString, nil
}

func DecodeAuthToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SIGNING_KEY")), nil
	})
	if err != nil {
		if err == jwt.ErrSignatureInvalid {
			err = errors.New("invalid token signature used")
			return nil, err
		}
		logger.Error("error decoding jwt", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil, err
	}
	if !token.Valid {
		err := errors.New("invalid token used")
		logger.Error(err.Error())
		return nil, err
	}
	return token, nil
}

func SignOutUser(ctx any, id string, reason string) {
	logger.Info("system user signout initiated", logger.LoggerOptions{
		Key:  "reason",
		Data: reason,
	})
	deleted := cache.Cache.DeleteOne(id)
	if !deleted {
		logger.Error("failed to sign out user", logger.LoggerOptions{
			Key:  "id",
			Data: id,
		})
	}
}
