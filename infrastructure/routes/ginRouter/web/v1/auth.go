package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func AuthRouter(router *gin.RouterGroup) {
	authRouter := router.Group("/auth")
	{
		authRouter.POST("/org/create", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.CreateOrgDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx, utils.GetStringPointer(ctx.GetHeader("Polymer-Device-Id")))
				return
			}
			controller.CreateOrganisation(&interfaces.ApplicationContext[dto.CreateOrgDTO]{
				Ctx:       ctx,
				Body:      &body,
				DeviceID:  appContext.DeviceID,
				UserAgent: appContext.UserAgent,
			})
		})

		authRouter.POST("/otp/verify", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.VerifyOTPDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx, utils.GetStringPointer(ctx.GetHeader("Polymer-Device-Id")))
				return
			}
			controller.VerifyOTP(&interfaces.ApplicationContext[dto.VerifyOTPDTO]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		authRouter.POST("/org/login", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.LoginDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx, utils.GetStringPointer(ctx.GetHeader("Polymer-Device-Id")))
				return
			}
			controller.LoginOrgMember(&interfaces.ApplicationContext[dto.LoginDTO]{
				Ctx:      ctx,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})

		authRouter.PATCH("/org/email/verify", middlewares.OTPTokenMiddleware("org_verification"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.VerifyOrg(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				DeviceID: appContext.DeviceID,
				Keys:     appContext.Keys,
			})
		})
	}
}
