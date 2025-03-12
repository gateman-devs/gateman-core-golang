package routev1

import (
	"encoding/json"
	"os"

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
			if os.Getenv("ENV") != "dev" {
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
			if os.Getenv("ENV") != "dev" {
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
			if os.Getenv("ENV") != "dev" {
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
			controller.LoginWorkspaceMember(&interfaces.ApplicationContext[dto.LoginWorkspaceMemberDTO]{
				Ctx:      ctx,
				Body:     &body,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
