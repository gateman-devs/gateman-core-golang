package middlewares

import (
	"fmt"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/interfaces"
	"gateman.io/application/utils"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
)

func DecryptPayloadMiddleware(ctx *interfaces.ApplicationContext[string]) []byte {
	if ctx.Body == nil || *ctx.Body == "" {
		return nil
	}
	sharedKey := cache.Cache.FindOne(fmt.Sprintf("%s-key", ctx.DeviceID))
	if sharedKey == nil {
		apperrors.ClientError(ctx.Ctx, "expired encryption key", nil, nil, ctx.DeviceID)
		return nil
	}

	decryptedKey, err := cryptography.DecryptData(*sharedKey, nil)
	if err != nil {
		logger.Error("an error occured while decrypting user payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	result, err := cryptography.DecryptData(*ctx.Body, utils.GetStringPointer(string(decryptedKey)))
	if err != nil {
		logger.Error("an error occured while decrypting user payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	return result
}
