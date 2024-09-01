package routev1

import (
	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/utils"
	"github.com/gin-gonic/gin"
)

func MiscRouter(router *gin.RouterGroup) {
	miscRouter := router.Group("/misc")
	{
		miscRouter.GET("/signedurl/generate", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.GeneratedSignedURLDTO
			if err := ctx.ShouldBindJSON(&body); err != nil {
				apperrors.ErrorProcessingPayload(ctx, utils.GetStringPointer(*appContext.DeviceID))
				return
			}
			controller.GeneratedSignedURL(&interfaces.ApplicationContext[dto.GeneratedSignedURLDTO]{
				Body: &body,
				Keys: appContext.Keys,
			})
		})
	}
}
