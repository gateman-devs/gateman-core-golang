package routev1

import (
	"os"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/entities"
	middlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func MiscRouter(router *gin.RouterGroup) {
	miscRouter := router.Group("/misc")
	{
		miscRouter.POST("/signedurl/generate", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GeneratedSignedURLDTO
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
			controller.GeneratedSignedURL(&interfaces.ApplicationContext[dto.GeneratedSignedURLDTO]{
				Body: &body,
				Keys: appContext.Keys,
				Ctx:  ctx,
			})
		})

		miscRouter.GET("/subscription-plans", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.GetSubscriptionPlans(&interfaces.ApplicationContext[any]{
				Keys: appContext.Keys,
				Ctx:  ctx,
			})
		})

		miscRouter.POST("/subscription/link", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_BILLING}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GeneratePaymentLinkDTO
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
			controller.GeneratePaymentLink(&interfaces.ApplicationContext[dto.GeneratePaymentLinkDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		miscRouter.POST("/card/add", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_BILLING}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GenerateAddCardLinkDTO
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
			controller.GenerateLinkToAddCard(&interfaces.ApplicationContext[dto.GenerateAddCardLinkDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})
	}
}
