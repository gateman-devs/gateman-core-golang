package v1

import (
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"

	appMiddlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func BiometricRouter(router *gin.RouterGroup) {
	biometricRouter := router.Group("/biometric")
	// Add activity logging middleware to all biometric routes
	biometricRouter.Use(middlewares.ActivityLogMiddleware())
	{
		biometricRouter.POST("/compare-faces", appMiddlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.EnhancedFaceComparisonRequest
			var deviceID string

			if os.Getenv("APP_ENV") != "dev" {
				// decryptedPayload, exists := ctx.Get("DecryptedBody")
				// if !exists {
				// 	apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
				// 	return
				// }
				// json.Unmarshal([]byte(decryptedPayload.(string)), &body)
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
				deviceID = appContext.DeviceID
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					deviceID := ctx.GetHeader("X-Device-Id")
					apperrors.ErrorProcessingPayload(ctx, &deviceID)
					return
				}
				deviceID = ctx.GetHeader("X-Device-Id")
			}
			controller.EnhancedFaceComparison(&interfaces.ApplicationContext[dto.EnhancedFaceComparisonRequest]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: deviceID,
				Keys:     appContext.Keys,
				Query: map[string]any{
					"fail": ctx.Query("fail"),
				},
			})
		})

		biometricRouter.POST("/liveness-check", appMiddlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.LivenessDetectionDTO
			var deviceID string

			if os.Getenv("APP_ENV") != "dev" {
				// decryptedPayload, exists := ctx.Get("DecryptedBody")
				// if !exists {
				// 	apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
				// 	return
				// }
				// json.Unmarshal([]byte(decryptedPayload.(string)), &body)
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
				deviceID = appContext.DeviceID
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					deviceID := ctx.GetHeader("X-Device-Id")
					apperrors.ErrorProcessingPayload(ctx, &deviceID)
					return
				}
				deviceID = ctx.GetHeader("X-Device-Id")
			}
			controller.EnhancedLivenessCheck(&interfaces.ApplicationContext[dto.LivenessDetectionDTO]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: deviceID,
				Keys:     appContext.Keys,
				Query: map[string]any{
					"fail": ctx.Query("fail"),
				},
			})
		})

		biometricRouter.GET("/generate-challenge", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.GenerateChallenge(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
			})
		})

		// // Verify video liveness endpoint
		biometricRouter.POST("/verify-video-liveness", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.VideoLivenessVerificationRequest
			if os.Getenv("APP_ENV") != "dev" {
				// decryptedPayload, exists := ctx.Get("DecryptedBody")
				// if !exists {
				// 	apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
				// 	return
				// }
				// json.Unmarshal([]byte(decryptedPayload.(string)), &body)
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
			} else {
				if err := ctx.ShouldBindJSON(&body); err != nil {
					apperrors.ErrorProcessingPayload(ctx, appContext.GetHeader("X-Device-Id"))
					return
				}
			}
			controller.VideoLivenessCheck(&interfaces.ApplicationContext[dto.VideoLivenessVerificationRequest]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
