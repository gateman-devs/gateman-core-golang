package routev1

import (
	"encoding/json"
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"github.com/gin-gonic/gin"
)

func BiometricRouter(router *gin.RouterGroup) {
	biometricRouter := router.Group("/biometric")
	{
		biometricRouter.POST("/compare-faces", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FaceComparisonRequest
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
			controller.CompareFaces(&interfaces.ApplicationContext[dto.FaceComparisonRequest]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		biometricRouter.POST("/liveness-check", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.LivenessCheckRequest
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
			controller.ImageLivenessCheck(&interfaces.ApplicationContext[dto.LivenessCheckRequest]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
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
			controller.VideoLivenessCheck(&interfaces.ApplicationContext[dto.VideoLivenessVerificationRequest]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		// Enhanced face comparison endpoint
		biometricRouter.POST("/enhanced-compare-faces", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.EnhancedFaceComparisonRequest
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
			controller.EnhancedFaceComparison(&interfaces.ApplicationContext[dto.EnhancedFaceComparisonRequest]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		// Enhanced liveness detection endpoint
		biometricRouter.POST("/enhanced-liveness-check", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.LivenessDetectionDTO
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
			controller.EnhancedLivenessCheck(&interfaces.ApplicationContext[dto.LivenessDetectionDTO]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		// Image quality check endpoint
		biometricRouter.POST("/image-quality-check", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ImageQualityDTO
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
			controller.ImageQualityCheck(&interfaces.ApplicationContext[dto.ImageQualityDTO]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		// System health check endpoint
		biometricRouter.GET("/system-health", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.SystemHealthCheck(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
