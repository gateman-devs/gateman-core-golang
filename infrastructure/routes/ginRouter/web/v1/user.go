package routev1

import (
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/interfaces"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func UserRouter(router *gin.RouterGroup) {
	userRouter := router.Group("/user")
	{
		userRouter.POST("/set-image", middlewares.UserAuthenticationMiddleware("face_verification"), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.SetAccountImage(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
