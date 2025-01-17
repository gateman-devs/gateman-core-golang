package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"github.com/gin-gonic/gin"
)

func OTPTokenMiddleware(intent string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		appContext, next := middlewares.OTPTokenMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     savedCtx.Keys,
			DeviceID: savedCtx.DeviceID,
			Header:   ctx.Request.Header,
		}, ctx.ClientIP(), intent)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
