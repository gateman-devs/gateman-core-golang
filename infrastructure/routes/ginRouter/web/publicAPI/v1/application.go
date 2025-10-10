package v1

import (
	"os"

	apperrors "gateman.io/application/appErrors"
	public_controller "gateman.io/application/controller/devAPI"
	"gateman.io/application/controller/devAPI/dto"
	"gateman.io/application/interfaces"
	middlewares "gateman.io/infrastructure/middleware"
	"github.com/gin-gonic/gin"
)

func AppRouter(router *gin.RouterGroup) {
	appRouter := router.Group("/app")
	{
		appRouter.POST("/fetch", middlewares.AppAuthenticationMiddleware(), func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			var body dto.FetchAppDTO
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
			accessToken, _ := ctx.Cookie("accessToken")
			public_controller.APIFetchAppDetails(&interfaces.ApplicationContext[dto.FetchAppDTO]{
				Ctx: ctx,
				Keys: map[string]any{
					"accessToken": accessToken,
				},
				Param:    appContext.Param,
				Body:     &body,
				DeviceID: appContext.DeviceID,
			})
		})
	}
}
