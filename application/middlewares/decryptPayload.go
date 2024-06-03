package middlewares

import (
	"encoding/hex"
	"fmt"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
)

func DecryptPayloadMiddleware(ctx *interfaces.ApplicationContext[string]) []byte {
	if ctx.Body == nil || *ctx.Body == "" {
		return nil
	}
	sharedKey := cache.Cache.FindOne(fmt.Sprintf("%s-key", *ctx.DeviceID))
	if sharedKey == nil {
		apperrors.ClientError(ctx.Ctx, "expired encryption key", nil, nil, nil)
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
	result, err := cryptography.DecryptData(*ctx.Body, utils.GetStringPointer(hex.EncodeToString(decryptedKey)))
	if err != nil {
		logger.Error("an error occured while decrypting user payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	return result
}
