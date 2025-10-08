package middlewares

import (
	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"github.com/gin-gonic/gin"
)

func RefreshTokenMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		workspaceToken := false
		refreshToken, _ := ctx.Cookie("refreshToken")
		if refreshToken == "" {
			refreshToken, _ = ctx.Cookie("workspaceRefreshToken")
			workspaceToken = true
		}
		appContext, next := middlewares.RefreshTokenMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, workspaceToken, refreshToken)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
