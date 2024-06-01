package middlewares

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
)

func DecryptPayloadMiddleware(ctx *interfaces.ApplicationContext[string]) *string {
	if ctx.Body == nil {
		return nil
	}
	sharedKey := cache.Cache.FindOne(fmt.Sprintf("%s-key", *ctx.DeviceID))
	if sharedKey == nil {
		apperrors.ClientError(ctx.Ctx, "expired encryption key", nil, nil, nil, nil)
		return nil
	}

	decryptedKey, err := cryptography.DecryptData(*sharedKey, nil, nil)
	if err != nil {
		logger.Error("an error occured while decrypting user payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}
	result, err := cryptography.DecryptData(*ctx.Body, utils.GetStringPointer(hex.EncodeToString(decryptedKey)), ctx.Nonce)
	if err != nil {
		logger.Error("an error occured while decrypting user payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return nil
	}

	fmt.Println("stop")
	var b any
	err = json.Unmarshal(result, &b)
	fmt.Println(err)
	fmt.Println(b)
	return utils.GetStringPointer(string(result))
}
