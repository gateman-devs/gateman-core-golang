package server_response

import (
	// "encoding/json"

	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"gateman.io/application/constants"
	"gateman.io/application/utils"
	"gateman.io/infrastructure/cryptography"
	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	"github.com/gin-gonic/gin"
)

type ginResponder struct{}

// Sends an encrypted payload to the client
func (gr ginResponder) Respond(ctx interface{}, code int, message string, payload any, errs []error, responseCode *uint, deviceID *string) {
	gr.UnEncryptedRespond(ctx, code, message, payload, errs, responseCode)
	return
	// if os.Getenv("APP_ENV") == "dev" || deviceID == nil {
	// 	gr.UnEncryptedRespond(ctx, code, message, payload, errs, responseCode)
	// 	return
	// }
	ginCtx, ok := (ctx).(*gin.Context)
	if !ok {
		logger.Error("could not transform *interface{} to gin.Context in serverResponse package", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx,
		})
		return
	}
	ginCtx.Abort()

	if payload != nil {
		switch p := payload.(type) {
		case map[string]any:
			if value, ok := p["accessToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:   "accessToken",
					Value:  *value.(*string),
					Domain: utils.ExtractDomain(utils.ExtractDomain(os.Getenv("CLIENT_URL"))),
					// HttpOnly: true,
					// Secure:   true,
					Path:     "/",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 1),
					MaxAge:   3600,
				})
			}
			if value, ok := p["refreshToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:   "refreshToken",
					Value:  *value.(*string),
					Domain: utils.ExtractDomain(os.Getenv("CLIENT_URL")),
					// HttpOnly: true,
					// Secure:   true,
					Path:     "/api/v1/auth/refresh",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 24 * 183),
					MaxAge:   15768000,
				})
			}
			if value, ok := p["workspaceAccessToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:   "workspaceAccessToken",
					Value:  *value.(*string),
					Domain: utils.ExtractDomain(os.Getenv("CLIENT_URL")),
					// HttpOnly: true,
					// Secure:   true,
					Path:     "/",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 1),
					MaxAge:   3600,
				})
			}
			if value, ok := p["workspaceRefreshToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:   "workspaceRefreshToken",
					Value:  *value.(*string),
					Domain: utils.ExtractDomain(os.Getenv("CLIENT_URL")),
					// HttpOnly: true,
					// Secure:   true,
					Path:     "/api/v1/auth/workspace/refresh",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 24 * 183),
					MaxAge:   15768000,
				})
			}
			delete(payload.(map[string]any), "accessToken")
			delete(payload.(map[string]any), "refreshToken")
			delete(payload.(map[string]any), "workspaceAccessToken")
			delete(payload.(map[string]any), "workspaceRefreshToken")
		}
	}

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

func (gr ginResponder) UnEncryptedRespond(ctx interface{}, code int, message string, payload any, errs []error, responseCode *uint) {
	ginCtx, ok := (ctx).(*gin.Context)
	if !ok {
		logger.Error("could not transform *interface{} to gin.Context in serverResponse package", logger.LoggerOptions{
			Key:  "payload",
			Data: ctx,
		})
		return
	}
	ginCtx.Abort()

	if payload != nil {
		switch p := payload.(type) {
		case map[string]any:
			if value, ok := p["accessToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:     "accessToken",
					Value:    *value.(*string),
					Domain:   utils.ExtractDomain(os.Getenv("AUTH_CLIENT_URL")),
					HttpOnly: true,
					Secure:   true,
					Path:     "/",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 1),
					MaxAge:   3600,
				})
			}
			if value, ok := p["refreshToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:     "refreshToken",
					Value:    *value.(*string),
					Domain:   utils.ExtractDomain(os.Getenv("AUTH_CLIENT_URL")),
					HttpOnly: true,
					Secure:   true,
					Path:     "/api/v1/auth/refresh",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 24 * 183),
					MaxAge:   15768000,
				})
			}
			if value, ok := p["workspaceAccessToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:     "workspaceAccessToken",
					Value:    *value.(*string),
					Domain:   utils.ExtractDomain(os.Getenv("WORKSPACE_CLIENT_URL")),
					HttpOnly: true,
					Secure:   true,
					Path:     "/",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 1),
					MaxAge:   3600,
				})
			}
			if value, ok := p["workspaceRefreshToken"]; ok && value.(*string) != nil {
				http.SetCookie(ginCtx.Writer, &http.Cookie{
					Name:     "workspaceRefreshToken",
					Value:    *value.(*string),
					Domain:   utils.ExtractDomain(os.Getenv("WORKSPACE_CLIENT_URL")),
					HttpOnly: true,
					Secure:   true,
					Path:     "/api/v1/auth/workspace/refresh",
					SameSite: http.SameSiteStrictMode,
					Expires:  time.Now().Add(time.Hour * 24 * 183),
					MaxAge:   15768000,
				})
			}
			delete(payload.(map[string]any), "accessToken")
			delete(payload.(map[string]any), "refreshToken")
			delete(payload.(map[string]any), "workspaceAccessToken")
			delete(payload.(map[string]any), "workspaceRefreshToken")
		}
	}

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
