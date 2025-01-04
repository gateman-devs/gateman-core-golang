package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"github.com/gin-gonic/gin"
)

func OTPTokenMiddleware(intent string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appContext, next := middlewares.OTPTokenMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
			Header:   ctx.Request.Header,
		}, ctx.ClientIP(), intent)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
