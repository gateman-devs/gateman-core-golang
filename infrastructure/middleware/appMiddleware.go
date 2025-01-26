package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"github.com/gin-gonic/gin"
)

func AppAuthenticationMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		appContext, next := middlewares.AppAuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     savedCtx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, ctx.ClientIP())
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
