package routev1

import (
	"os"
	"strconv"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/entities"
	middlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func WorkspaceRouter(router *gin.RouterGroup) {
	workspaceRouter := router.Group("/workspace")
	{
		workspaceRouter.POST("/create", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.CreateWorkspaceDTO
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
			controller.CreateWorkspace(&interfaces.ApplicationContext[dto.CreateWorkspaceDTO]{
				Ctx:        ctx,
				Body:       &body,
				DeviceID:   appContext.DeviceID,
				DeviceName: appContext.DeviceName,
				UserAgent:  appContext.UserAgent,
				Keys:       appContext.Keys,
				Param: map[string]any{
					"ip": ctx.ClientIP(),
				},
			})
		})

		workspaceRouter.POST("/invite", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.MEMBER_INVITE}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.InviteWorspaceMembersDTO
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
			controller.InviteWorkspaceMembers(&interfaces.ApplicationContext[dto.InviteWorspaceMembersDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		workspaceRouter.POST("/verify", middlewares.OTPTokenMiddleware("verify_workspace"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.VerifyWorkspaceAccount(&interfaces.ApplicationContext[any]{
				Ctx:        ctx,
				Keys:       appContext.Keys,
				DeviceID:   appContext.DeviceID,
				DeviceName: appContext.DeviceName,
				Param: map[string]any{
					"ip": ctx.ClientIP(),
				},
			})
		})

		workspaceRouter.POST("/login", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.LoginWorkspaceMemberDTO
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
			controller.LoginWorkspaceMember(&interfaces.ApplicationContext[dto.LoginWorkspaceMemberDTO]{
				Ctx:      ctx,
				Body:     &body,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
				Param: map[string]any{
					"ip": ctx.ClientIP(),
				},
			})
		})

		workspaceRouter.POST("/help", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.WORKSPACE_CREATE_APPLICATIONS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.CreateHelpRequestDTO
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
			controller.CreateHelpRequest(&interfaces.ApplicationContext[dto.CreateHelpRequestDTO]{
				Ctx:  ctx,
				Body: &body,
				Keys: appContext.Keys,
			})
		})

		workspaceRouter.GET("/help", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.SUPER_ACCESS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchHelpRequestsDTO

			// Parse query parameters
			if limitStr := ctx.Query("limit"); limitStr != "" {
				if limit, err := strconv.ParseInt(limitStr, 10, 64); err == nil {
					body.Limit = limit
				} else {
					body.Limit = 10 // default
				}
			} else {
				body.Limit = 10 // default
			}

			if lastID := ctx.Query("lastID"); lastID != "" {
				body.LastID = &lastID
			}

			if status := ctx.Query("status"); status != "" {
				body.Status = &status
			}

			controller.FetchHelpRequests(&interfaces.ApplicationContext[dto.FetchHelpRequestsDTO]{
				Ctx:      ctx,
				Body:     &body,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})

		workspaceRouter.GET("/fetch", middlewares.WorkspaceAuthenticationMiddleware(nil, &[]entities.MemberPermissions{entities.SUPER_ACCESS}, true), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])

			controller.FetchWorkspaceDetails(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
