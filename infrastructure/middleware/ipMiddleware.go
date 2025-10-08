package middlewares

import (
	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
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
