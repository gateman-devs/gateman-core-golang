package middlewares

import (
	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"github.com/gin-gonic/gin"
)

func RefreshTokenMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		refreshToken, _ := ctx.Cookie("accessToken")
		appContext, next := middlewares.RefreshTokenMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, refreshToken)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
