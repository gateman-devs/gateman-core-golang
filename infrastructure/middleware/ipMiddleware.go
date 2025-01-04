package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"github.com/gin-gonic/gin"
)

func IPAddressMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		appContext, next := middlewares.IPAddressMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:        ctx,
			Keys:       savedCtx.Keys,
			Header:     ctx.Request.Header,
			DeviceID:   savedCtx.DeviceID,
			DeviceName: savedCtx.DeviceName,
		}, ctx.ClientIP())
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
