package routev1

import (
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/utils"
	middlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func UserRouter(router *gin.RouterGroup) {
	userRouter := router.Group("/user")
	{
		userRouter.POST("/set-image", middlewares.UserAuthenticationMiddleware(utils.GetStringPointer("face_verification")), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.SetAccountImage(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})

		userRouter.POST("/set-nin", middlewares.UserAuthenticationMiddleware(nil), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SetNINDetails
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
			controller.SetNINDetails(&interfaces.ApplicationContext[dto.SetNINDetails]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
				Body:     &body,
			})
		})

		userRouter.POST("/drivers-license", middlewares.UserAuthenticationMiddleware(nil), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SetDriversLicenseDetails
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
			controller.SetDriversLicenseDetails(&interfaces.ApplicationContext[dto.SetDriversLicenseDetails]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
				Body:     &body,
			})
		})

		userRouter.POST("/verify-nin", middlewares.OTPTokenMiddleware("verify_nin"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.VerifyNINDetails(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})

		userRouter.POST("/set-bvn", middlewares.UserAuthenticationMiddleware(nil), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SetBVNDetails
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
			controller.SetBVNDetails(&interfaces.ApplicationContext[dto.SetBVNDetails]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
				Body:     &body,
			})
		})

		userRouter.POST("/verify-bvn", middlewares.OTPTokenMiddleware("verify_bvn"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.VerifyBVNDetails(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})

		userRouter.POST("/set-voter-id", middlewares.UserAuthenticationMiddleware(nil), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SetVoterIDDetails
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
			controller.SetVoterIDDetails(&interfaces.ApplicationContext[dto.SetVoterIDDetails]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
				Body:     &body,
			})
		})

		userRouter.POST("/verify-voter-id", middlewares.OTPTokenMiddleware("verify_voter_id"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.VerifyVoterIDDetails(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
