package server_response

import (
	"authone.usepolymer.co/infrastructure/logger"
	"github.com/gin-gonic/gin"
)

type ginResponder struct{}

// Sends an encrypted payload to the client
func (gr ginResponder) Respond(ctx interface{}, code int, message string, payload interface{}, errs []error, responseCode *uint, accessToken *string, refreshToken *string) {
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
	if accessToken != nil {
		response["accessToken"] = *accessToken
	}
	if refreshToken != nil {
		response["refreshToken"] = *refreshToken
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
