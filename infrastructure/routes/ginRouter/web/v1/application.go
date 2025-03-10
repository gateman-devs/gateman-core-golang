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
			var body dto.UpdateApplications
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.UpdateApplication(&interfaces.ApplicationContext[dto.UpdateApplications]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
				Header: ctx.Request.Header,
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

		appRouter.PATCH("/sandbox-apikey/refresh/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshSandboxAppAPIKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/app-signing-key/refresh/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshAppSigningKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/sandbox-app-signing-key/refresh/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshSandboxAppSigningKey(&interfaces.ApplicationContext[any]{
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

		appRouter.POST("/signup", middlewares.UserAuthenticationMiddleware("", nil, false), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ApplicationSignUpDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			appContext.Keys["ip"] = ctx.ClientIP()
			controller.ApplicationSignUp(&interfaces.ApplicationContext[dto.ApplicationSignUpDTO]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
			})
		})

		appRouter.POST("/users", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{
			entities.USER_VIEW,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchAppUsersDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.FetchAppUsers(&interfaces.ApplicationContext[dto.FetchAppUsersDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.PATCH("/users/block/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{
			entities.USER_BLOCK,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.BlockAccountsDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.BlockAccounts(&interfaces.ApplicationContext[dto.BlockAccountsDTO]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/users/unblock/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{
			entities.USER_BLOCK,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.BlockAccountsDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.UnblockAccounts(&interfaces.ApplicationContext[dto.BlockAccountsDTO]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/ttl/update/:id", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{
			entities.USER_BLOCK,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.UpdateAccessRefreshTokenTTL
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.UpdateAccessRefreshTokenTTL(&interfaces.ApplicationContext[dto.UpdateAccessRefreshTokenTTL]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.POST("/metrics", middlewares.UserAuthenticationMiddleware("", &[]entities.MemberPermissions{
			entities.WORKSPACE_VIEW_APPLICATIONS,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchAppMetrics
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.GetAppMetrics(&interfaces.ApplicationContext[dto.FetchAppMetrics]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
			})
		})
	}
}
