package middlewares

import (
	"os"

	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"github.com/gin-gonic/gin"
)

func UserAgentMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appContext, next := middlewares.UserAgentMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, os.Getenv("MIN_CLIENT_VERSION"), ctx.ClientIP())
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
