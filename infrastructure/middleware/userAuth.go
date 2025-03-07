package middlewares

import (
	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"github.com/gin-gonic/gin"
)

func UserAuthenticationMiddleware(intent *string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		accessToken, _ := ctx.Cookie("accessToken")
		appContext, next := middlewares.UserAuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     savedCtx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, intent, accessToken)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
