package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	middlewares "authone.usepolymer.co/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func OrgRouter(router *gin.RouterGroup) {
	orgRouter := router.Group("/workspace")
	{
		orgRouter.POST("/create", middlewares.UserAuthenticationMiddleware("", nil, false), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.CreateOrgDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			controller.CreateOrganisation(&interfaces.ApplicationContext[dto.CreateOrgDTO]{
				Ctx:       ctx,
				Body:      &body,
				DeviceID:  appContext.DeviceID,
				UserAgent: appContext.UserAgent,
				Keys:      appContext.Keys,
			})
		})

		orgRouter.GET("/fetch", middlewares.UserAuthenticationMiddleware("", nil, false), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			controller.FetchWorkspaces(&interfaces.ApplicationContext[any]{
				Ctx:      ctx,
				Keys:     appContext.Keys,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
