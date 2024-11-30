package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func UserRouter(router *gin.RouterGroup) {
	userRouter := router.Group("/user")
	{
		userRouter.POST("/set-image", middlewares.UserAuthenticationMiddleware("face_verification", nil, false), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.SetAccountImage(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})

		userRouter.POST("/set-nin", middlewares.UserAuthenticationMiddleware("", nil, false), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.SetNINDetails
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.SetNINDetails(&interfaces.ApplicationContext[dto.SetNINDetails]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
				Body:     &body,
			})
		})
	}
}
