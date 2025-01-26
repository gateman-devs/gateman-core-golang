package v1

import (
	public_controller "authone.usepolymer.co/application/controller/devAPI"
	"authone.usepolymer.co/application/interfaces"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func AppRouter(router *gin.RouterGroup) {
	appRouter := router.Group("/app")
	{
		appRouter.GET("/fetch/:id", middlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			public_controller.APIFetchAppDetails(&interfaces.ApplicationContext[any]{
				Ctx:  ctx,
				Keys: appContext.Keys,
				Param: map[string]any{
					"id": ctx.Param("id"),
				},
			})
		})
	}
}
