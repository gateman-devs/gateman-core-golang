package server_response

import (
	// "encoding/json"

	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/infrastructure/cryptography"
	"authone.usepolymer.co/infrastructure/database/repository/cache"
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/gin-gonic/gin"
)

type ginResponder struct{}

// Sends an encrypted payload to the client
func (gr ginResponder) Respond(ctx interface{}, code int, message string, payload interface{}, errs []error, response_code *uint, device_id *string) {
	ginCtx, ok := (ctx).(*gin.Context)
	if !ok {
		logger.Error("could not transform *interface{} to gin.Context in serverResponse package", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx,
		})
		return
	}
	ginCtx.Abort()
	response := map[string]any{
		"message": message,
		"body":    payload,
	}
	if response_code != nil {
		response["response_code"] = response_code
	}
	if os.Getenv("ENV") != "prod" {
		logger.Info("response", logger.LoggerOptions{
			Key:  "message",
			Data: message,
		}, logger.LoggerOptions{
			Key:  "error",
			Data: errs,
		})
	}
	if errs != nil {
		errMsgs := []string{}
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		response["errors"] = errMsgs
	}
	device_id = nil
	if device_id == nil {
		ginCtx.JSON(code, response)
		return
	}
	jsonResponse, _ := json.Marshal(response)
	enc_key := cache.Cache.FindOne(fmt.Sprintf("%s-key", *device_id))
	if enc_key == nil {
		ginCtx.JSON(401, map[string]any{
			"response_code": constants.ENCRYPTION_KEY_EXPIRED,
			"message":       "encryption key has expired. initiate key exchange protocol again.",
		})
		return
	}
	decryptedKey, _ := cryptography.DecryptData(*enc_key, nil)
	encryptedResponse, err := cryptography.EncryptData(jsonResponse, utils.GetStringPointer(hex.EncodeToString(decryptedKey)))
	if err != nil {
		logger.Error("error encrypting data", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
	}
	ginCtx.JSON(code, encryptedResponse)
	ginCtx, ok = (ctx).(*gin.Context)
	if !ok {
		logger.Error("could not transform *interface{} to gin.Context in serverResponse package", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx,
		})
		return
	}
	ginCtx.Abort()
}

// Sends a response to the client using plain JSON
func (gr ginResponder) UnEncryptedRespond(ctx interface{}, code int, message string, payload interface{}, errs []error, response_code *uint) {
	ginCtx, ok := (ctx).(*gin.Context)
	if !ok {
		logger.Error("could not transform *interface{} to gin.Context in serverResponse package", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx,
		})
		return
	}
	ginCtx.Abort()
	response := map[string]any{
		"message": message,
		"body":    payload,
	}
	if response_code != nil {
		response["response_code"] = response_code
	}
	if errs != nil {
		errMsgs := []string{}
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		response["errors"] = errMsgs
	}
	ginCtx.JSON(code, response)
}
