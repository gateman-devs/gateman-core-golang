package server_response

import (
	// "encoding/json"

	"encoding/json"
	"fmt"

	"gateman.io/application/constants"
	"gateman.io/application/utils"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"github.com/gin-gonic/gin"
)

type ginResponder struct{}

// Sends an encrypted payload to the client
func (gr ginResponder) Respond(ctx interface{}, code int, message string, payload interface{}, errs []error, responseCode *uint, deviceID *string) {
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
	if responseCode != nil {
		response["responseCode"] = responseCode
	}
	if errs != nil {
		errMsgs := []string{}
		for _, err := range errs {
			errMsgs = append(errMsgs, err.Error())
		}
		response["errors"] = errMsgs
	}
	if deviceID == nil {
		ginCtx.JSON(code, response)
		return
	}
	jsonResponse, _ := json.Marshal(response)

	sharedKey := cache.Cache.FindOne(fmt.Sprintf("%s-key", *deviceID))
	if sharedKey == nil {
		ginCtx.JSON(401, map[string]any{
			"responseCode": constants.ENCRYPTION_KEY_EXPIRED,
			"message":      "encryption key has expired. initiate key exchange protocol again.",
		})
		return
	}
	decryptedKey, _ := cryptography.DecryptData(*sharedKey, nil)
	encryptedResponse, err := cryptography.EncryptData(jsonResponse, utils.GetStringPointer(string(decryptedKey)))
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
func (gr ginResponder) UnEncryptedRespond(ctx interface{}, code int, message string, payload interface{}, errs []error, responseCode *uint) {
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
	if responseCode != nil {
		response["responseCode"] = responseCode
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
