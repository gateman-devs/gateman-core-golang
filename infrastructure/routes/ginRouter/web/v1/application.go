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

func AppRouter(router *gin.RouterGroup) {
	appRouter := router.Group("/app")
	{
		appRouter.POST("/create", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_CREATE_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ApplicationDTO
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
			controller.CreateApplication(&interfaces.ApplicationContext[dto.ApplicationDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.PATCH("/ip-whitelist/update/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_CREATE_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.UpdateWhitelistIPDTO
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
			controller.UpdateWhiteListedIPs(&interfaces.ApplicationContext[dto.UpdateWhitelistIPDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.PATCH("/update/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.UpdateApplications
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

		appRouter.PATCH("/apikey/refresh/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshAppAPIKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/sandbox-apikey/refresh/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshSandboxAppAPIKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/app-signing-key/refresh/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshAppSigningKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.PATCH("/sandbox-app-signing-key/refresh/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_EDIT_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.RefreshSandboxAppSigningKey(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})

		appRouter.GET("/workspace/fetch", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_VIEW_APPLICATIONS}, true), func(ctx *gin.Context) {
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
				apperrors.ClientError(ctx, "missing parameter id", nil, nil, *appContext.GetHeader("X-Device-Id"))
				return
			}
			accessToken, _ := ctx.Cookie("accessToken")
			controller.FetchAppDetails(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
				Keys: map[string]any{
					"ip":          ctx.ClientIP(),
					"accessToken": accessToken,
				},
				Header: appContext.Header,
				Param: map[string]any{
					"id": id,
				},
				DeviceID: appContext.DeviceID,
			})
		})

		appRouter.DELETE("/delete/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_DELETE_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			id, found := ctx.Params.Get("id")
			if !found {
				apperrors.ClientError(ctx, "missing parameter id", nil, nil, *appContext.GetHeader("X-Device-Id"))
			}
			controller.DeleteApplication(&interfaces.ApplicationContext[any]{
				Ctx:    ctx,
				Header: appContext.Header,
				Param: map[string]any{
					"id": id,
				},
			})
		})

		appRouter.GET("/config/fetch", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_CREATE_APPLICATIONS}, true), func(ctx *gin.Context) {
			controller.FetchAppCreationConfigInfo(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
			},
			)
		})

		appRouter.POST("/signup", middlewares.UserAuthenticationMiddleware(nil), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ApplicationSignUpDTO
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
			appContext.Keys["ip"] = ctx.ClientIP()
			controller.ApplicationSignUp(&interfaces.ApplicationContext[dto.ApplicationSignUpDTO]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
			})
		})

		appRouter.POST("/users", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.USER_VIEW,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchAppUsersDTO
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
			controller.FetchAppUsers(&interfaces.ApplicationContext[dto.FetchAppUsersDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.POST("/mfa/toggle", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.USER_VIEW,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.ToggleMFAProtectionSettingDTO
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
			controller.ToggleMFAProtectionSetting(&interfaces.ApplicationContext[dto.ToggleMFAProtectionSettingDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		appRouter.PATCH("/users/block/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.USER_BLOCK,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.BlockAccountsDTO
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

		appRouter.PATCH("/users/unblock/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.USER_BLOCK,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.BlockAccountsDTO
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

		appRouter.PATCH("/ttl/update/:id", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.WORKSPACE_EDIT_APPLICATIONS,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.UpdateAccessRefreshTokenTTL
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

		appRouter.POST("/metrics", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.WORKSPACE_VIEW_APPLICATIONS,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchAppMetrics
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
			controller.GetAppMetrics(&interfaces.ApplicationContext[dto.FetchAppMetrics]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
			})
		})

		appRouter.POST("/custom-form/submit", middlewares.UserAuthenticationMiddleware(nil), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SubmitCustomAppFormDTO
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
			controller.SubmitCustomAppForm(&interfaces.ApplicationContext[dto.SubmitCustomAppFormDTO]{
				Ctx:    ctx,
				Body:   &body,
				Keys:   appContext.Keys,
				Header: ctx.Request.Header,
			})
		})

		appRouter.POST("/activity-logs", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{
			entities.WORKSPACE_VIEW_APPLICATIONS,
		}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchActivityLogsDTO
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
			controller.FetchActivityLogs(&interfaces.ApplicationContext[dto.FetchActivityLogsDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})
	}
}
