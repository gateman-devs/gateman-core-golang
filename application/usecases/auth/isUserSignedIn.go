package auth_usecases

import (
	"fmt"
	"os"

	"gateman.io/infrastructure/auth"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"github.com/golang-jwt/jwt"
)

// UserAuthResult represents the result of user authentication
type UserAuthResult struct {
	IsAuthenticated bool
	UserID          string
	Email           string
	Phone           string
	UserAgent       string
	DeviceID        string
	ErrorMessage    string
}

// IsUserSignedIn validates if a user is properly authenticated
// Returns UserAuthResult with authentication status and user information
func IsUserSignedIn(ctx any, authToken any, intent *string, deviceID string) UserAuthResult {
	result := UserAuthResult{
		IsAuthenticated: false,
	}

	// Check if auth token is provided
	if authToken == "" || authToken == nil {
		result.ErrorMessage = "missing auth token"
		return result
	}

	// Decode and validate the JWT token
	validAccessToken, err := auth.DecodeAuthToken(authToken.(string))
	if err != nil {
		result.ErrorMessage = "this session has expired"
		return result
	}

	if !validAccessToken.Valid {
		result.ErrorMessage = "unauthorised access"
		return result
	}

	// Extract claims from the token
	authTokenClaims := validAccessToken.Claims.(jwt.MapClaims)

	// Validate issuer
	if authTokenClaims["iss"] != os.Getenv("GATEMAN_ISSUER") {
		logger.Warning("attempt to access account with tampered jwt", logger.LoggerOptions{
			Key:  "token claims",
			Data: validAccessToken,
		})
		result.ErrorMessage = "unauthorised access"
		return result
	}

	// Validate device ID matches the one in token
	if authTokenClaims["deviceID"] != deviceID {
		logger.Warning("client made request using device id different from that in access token", logger.LoggerOptions{
			Key:  "token device id",
			Data: authTokenClaims["deviceID"],
		}, logger.LoggerOptions{
			Key:  "request device id",
			Data: deviceID,
		})
		result.ErrorMessage = "unauthorized access"
		return result
	}

	// Validate token in cache
	deviceIDHash, _ := cryptography.CryptoHahser.HashString(deviceID, []byte(os.Getenv("HASH_FIXED_SALT")))
	validToken := cache.Cache.FindOne(fmt.Sprintf("%s-access", string(deviceIDHash)))
	if validToken == nil {
		result.ErrorMessage = "this session has expired"
		return result
	}

	// Verify token hash matches
	match := cryptography.CryptoHahser.VerifyHashData(*validToken, authToken.(string))
	if !match {
		result.ErrorMessage = "this session has expired"
		return result
	}

	// Validate intent if provided
	if intent != nil {
		if authTokenClaims["intent"] != *intent {
			result.ErrorMessage = "unauthorised access"
			return result
		}
	}

	// Validate account verification status
	if !authTokenClaims["verifiedAccount"].(bool) {
		result.ErrorMessage = "verify your account before trying to use this route"
		return result
	}

	// Validate token type
	if authTokenClaims["tokenType"] != "access_token" {
		result.ErrorMessage = "unauthorised access"
		return result
	}

	// Extract user information from token claims
	result.IsAuthenticated = true
	result.UserID = authTokenClaims["userID"].(string)
	result.Email = authTokenClaims["email"].(string)
	result.Phone = authTokenClaims["phone"].(string)
	result.UserAgent = authTokenClaims["userAgent"].(string)
	result.DeviceID = authTokenClaims["deviceID"].(string)

	return result
}
