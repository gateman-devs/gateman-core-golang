package middlewares

import (
	"context"
	"fmt"
	"strings"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/connection/cache"
)

func AppAuthenticationMiddleware(ctx *interfaces.ApplicationContext[any], ipAddress string) (*interfaces.ApplicationContext[any], bool) {
	apiKeyPointer := ctx.GetHeader("X-Api-Key")
	if apiKeyPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an api key", ctx.DeviceID)
		return nil, false
	}
	apiKey := *apiKeyPointer
	appIDPointer := ctx.GetHeader("X-App-Id")
	if appIDPointer == nil {
		apperrors.AuthenticationError(ctx.Ctx, "provide an app id", ctx.DeviceID)
		return nil, false
	}
	appID := *appIDPointer

	// Rate limit: 100 requests per minute per app ID
	if !CheckRateLimit(appID, time.Minute, 100) {
		apperrors.ClientError(ctx.Ctx, "rate limit exceeded, please try again later", nil, nil, ctx.DeviceID)
		return nil, false
	}
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": appID,
	})
	if app == nil {
		apperrors.NotFoundError(ctx.Ctx, "invalid credentials", ctx.GetHeader("X-Device-Id"))
		return nil, false
	}
	var appAPIKey string
	var sanbox = strings.Contains(apiKey, "sandbox")
	if sanbox {
		appAPIKey = app.SandboxAPIKey
		apiKey = strings.Split(apiKey, "-")[1]
	} else {
		appAPIKey = app.APIKey
	}
	match := cryptography.CryptoHahser.VerifyHashData(appAPIKey, apiKey)
	if !match {
		apperrors.ClientError(ctx.Ctx, "invalid credentials", nil, nil, ctx.DeviceID)
		return nil, false
	}
	// if os.Getenv("APP_ENV") == "production" && app.WhiteListedIPs == nil {
	// 	apperrors.ClientError(ctx.Ctx, "no ip address whitelisted", nil, nil, ctx.DeviceID)
	// 	return nil, false
	// }
	// if os.Getenv("APP_ENV") == "production" {
	// 	validIP := false
	// 	for _, wIP := range *app.WhiteListedIPs {
	// 		if wIP == ipAddress {
	// 			validIP = true
	// 			break
	// 		}
	// 	}
	// 	if !validIP {
	// 		apperrors.ClientError(ctx.Ctx, "unauthorised access", nil, nil, ctx.DeviceID)
	// 		return nil, false
	// 	}
	// }
	ctx.SetContextData("AppID", appID)
	ctx.SetContextData("Name", app.Name)
	ctx.SetContextData("WorkspaceID", app.WorkspaceID)
	ctx.SetContextData("SandboxEnv", sanbox)
	return ctx, true
}

// CheckRateLimit checks if the given ID has exceeded the rate limit using Redis
// id: unique identifier (e.g., IP address, user ID, app ID)
// timeWindow: duration of the rate limit window (e.g., time.Minute)
// maxRequests: maximum number of requests allowed in the time window
// Returns true if the request is allowed, false if rate limit is exceeded
func CheckRateLimit(id string, timeWindow time.Duration, maxRequests int) bool {
	redisClient, err := cache.GetInstance()
	if err != nil {
		// If Redis is unavailable, allow the request (fail open)
		return true
	}

	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:%s", id)

	// Use Redis INCR with expiry for atomic rate limiting
	pipe := redisClient.Client.Pipeline()

	// Increment the counter
	incrCmd := pipe.Incr(ctx, key)

	// Set expiry only if this is the first request (key didn't exist)
	pipe.Expire(ctx, key, timeWindow)

	// Execute pipeline
	_, err = pipe.Exec(ctx)
	if err != nil {
		// If Redis operation fails, allow the request (fail open)
		return true
	}

	// Get the current count
	count := incrCmd.Val()

	// Check if limit is exceeded
	return count <= int64(maxRequests)
}
