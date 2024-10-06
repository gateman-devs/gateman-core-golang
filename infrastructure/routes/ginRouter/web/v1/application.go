package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func AppRouter(router *gin.RouterGroup) {
	appRouter := router.Group("/app")
	{
		appRouter.POST("/create", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.ORG_CREATE_APPLICATIONS}), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ApplicationDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx, utils.GetStringPointer(ctx.GetHeader("Polymer-Device-Id")))
				return
			}
			controller.CreateApplication(&interfaces.ApplicationContext[dto.ApplicationDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.GET("/details/:id", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			id, found := ctx.Params.Get("id")
			if !found {
				apperrors.ClientError(ctx, "missing parameter id", nil, nil, utils.GetStringPointer(ctx.GetHeader("Polymer-Device-Id")))
			}
			controller.FetchAppDetails(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
				Keys: map[string]any{
					"ip": ctx.ClientIP(),
				},
				Header: appContext.Header,
				Param: map[string]any{
					"id": id,
				},
			})
		})

		appRouter.GET("/config/fetch", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.ORG_CREATE_APPLICATIONS}), func(ctx *gin.Context) {
			controller.FetchAppCreationConfigInfo(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
			},
			)
		})
	}
}
