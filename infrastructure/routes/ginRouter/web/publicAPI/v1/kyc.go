package v1

import (
	public_controller "gateman.io/application/controller/devAPI"
	"gateman.io/application/interfaces"
	middlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func KYCRouter(router *gin.RouterGroup) {
	kycRouter := router.Group("/kyc")
	{
		kycRouter.GET("/nin/:id", middlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			public_controller.APIFetchNINDetails(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
				Param: map[string]any{
					"nin": ctx.Param("id"),
				},
				DeviceID: appContext.DeviceID,
			})
		})
		kycRouter.GET("/bvn/:id", middlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			public_controller.APIFetchBVNDetails(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
				Param: map[string]any{
					"bvn": ctx.Param("id"),
				},
				DeviceID: appContext.DeviceID,
			})
		})

		kycRouter.GET("/voters-card/:id", middlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			public_controller.APIFetchVotersCardDetails(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
				Param: map[string]any{
					"votersID": ctx.Param("id"),
				},
				DeviceID: appContext.DeviceID,
			})
		})

		kycRouter.GET("/drivers-license/:id", middlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			public_controller.APIFetchDriversLicenseDetails(&interfaces.ApplicationContext[any]{
				Ctx: ctx,
				Param: map[string]any{
					"driversLicense": ctx.Param("id"),
				},
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
