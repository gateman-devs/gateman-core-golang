package routev1

import (
	"encoding/json"
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	middlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func AuthRouter(router *gin.RouterGroup) {
	authRouter := router.Group("/auth")
	{
		authRouter.POST("/otp/verify", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.VerifyOTPDTO
			if os.Getenv("APP_ENV") != "dev" {
				decryptedPayload, exists := ctx.Get("DecryptedBody")
				if !exists {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
				json.Unmarshal([]byte(decryptedPayload.(string)), &body)
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
			}
			controller.VerifyOTP(&interfaces.ApplicationContext[dto.VerifyOTPDTO]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		authRouter.PATCH("/user/account/verify", middlewares.OTPTokenMiddleware("verify_account"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])

			controller.VerifyUserAccount(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
				Keys:     appContext.Keys,
			})
		})

		authRouter.POST("/verify-device", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.VerifyDeviceDTO
			if os.Getenv("APP_ENV") != "dev" {
				decryptedPayload, exists := ctx.Get("DecryptedBody")
				if !exists {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
				json.Unmarshal([]byte(decryptedPayload.(string)), &body)
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
			}
			controller.VeirfyDeviceImage(&interfaces.ApplicationContext[dto.VerifyDeviceDTO]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
				Keys:     appContext.Keys,
				Body:     &body,
			})
		})

		authRouter.GET("/refresh", middlewares.RefreshTokenMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshToken(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
				Keys:     appContext.Keys,
			})
		})

		authRouter.GET("/workspace/refresh", middlewares.RefreshTokenMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.WorkspaceRefreshToken(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
				Keys:     appContext.Keys,
			})
		})

		authRouter.POST("/user/authenticate", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.CreateUserDTO
			if os.Getenv("APP_ENV") != "dev" {
				decryptedPayload, exists := ctx.Get("DecryptedBody")
				if !exists {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
				json.Unmarshal([]byte(decryptedPayload.(string)), &body)
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
			}
			body.UserAgent = *appContext.GetHeader("User-Agent")
			body.DeviceID = *appContext.GetHeader("X-Device-Id")
			body.DeviceName = appContext.DeviceName
			controller.AuthenticateUser(&interfaces.ApplicationContext[dto.CreateUserDTO]{
				Ctx:        ctx,
				Body:       &body,
				Keys:       appContext.Keys,
				DeviceID:   appContext.DeviceID,
				DeviceName: appContext.DeviceName,
				UserAgent:  appContext.UserAgent,
			})
		})

		authRouter.POST("/otp/resend", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ResendOTPDTO
			if os.Getenv("APP_ENV") != "dev" {
				decryptedPayload, exists := ctx.Get("DecryptedBody")
				if !exists {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
				json.Unmarshal([]byte(decryptedPayload.(string)), &body)
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
			}
			controller.ResendOTP(&interfaces.ApplicationContext[dto.ResendOTPDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})
	}
}
