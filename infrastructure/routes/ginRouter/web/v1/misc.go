package routev1

import (
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
		miscRouter.GET("/signedurl/generate", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GeneratedSignedURLDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
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

		miscRouter.POST("/subscription/link", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_BILLING}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GeneratePaymentLinkDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.GeneratePaymentLink(&interfaces.ApplicationContext[dto.GeneratePaymentLinkDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		miscRouter.POST("/card/add", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_BILLING}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GenerateAddCardLinkDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.GenerateLinkToAddCard(&interfaces.ApplicationContext[dto.GenerateAddCardLinkDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})
	}
}
