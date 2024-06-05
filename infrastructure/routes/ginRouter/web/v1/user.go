package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	"github.com/gin-gonic/gin"
)

func UserRouter(router *gin.RouterGroup) {
	userRouter := router.Group("/user")
	{
		userRouter.POST("/create", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.CreateUserDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx, utils.GetStringPointer(ctx.GetHeader("Polymer-Device-Id")))
				return
			}
			controller.CreateUser(&interfaces.ApplicationContext[dto.CreateUserDTO]{
				Ctx:       ctx,
				Body:      &body,
				Keys:      appContext.Keys,
				DeviceID:  appContext.DeviceID,
				UserAgent: appContext.UserAgent,
			})
		})
	}
}
