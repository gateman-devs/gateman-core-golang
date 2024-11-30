package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"authone.usepolymer.co/application/utils"
	"github.com/gin-gonic/gin"
)

func RefreshTokenMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appContext, next := middlewares.RefreshTokenMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: utils.GetStringPointer(ctx.Request.Header.Get("X-Device-Id")),
		})
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
