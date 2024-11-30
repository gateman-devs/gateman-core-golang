package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/entities"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func AppRouter(router *gin.RouterGroup) {
	appRouter := router.Group("/app")
	{
		appRouter.POST("/create", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_CREATE_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ApplicationDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.CreateApplication(&interfaces.ApplicationContext[dto.ApplicationDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.PATCH("/update/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ApplicationDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.UpdateApplication(&interfaces.ApplicationContext[dto.ApplicationDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/apikey/refresh/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshAppAPIKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.GET("/workspace/fetch", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_VIEW_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.FetchWorkspaceApps(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
			})
		})

		appRouter.GET("/details/:id", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			id, found := ctx.Params.Get("id")
			if !found {
				apperrors.ClientError(ctx, "missing parameter id", nil, nil)
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

		appRouter.DELETE("/delete/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_DELETE_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			id, found := ctx.Params.Get("id")
			if !found {
				apperrors.ClientError(ctx, "missing parameter id", nil, nil)
			}
			controller.DeleteApplication(&interfaces.ApplicationContext[any]{
				Ctx:    ctx,
				Header: appContext.Header,
				Param: map[string]any{
					"id": id,
				},
			})
		})

		appRouter.GET("/config/fetch", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_CREATE_APPLICATIONS}, true), func(ctx *gin.Context) {
			controller.FetchAppCreationConfigInfo(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
			},
			)
		})

		appRouter.GET("/all", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_VIEW_APPLICATIONS}, true), func(ctx *gin.Context) {
			controller.FetchWorkspaceApps(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
			},
			)
		})
	}
}
