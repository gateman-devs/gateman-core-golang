package middlewares

import (
	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"gateman.io/entities"
	"github.com/gin-gonic/gin"
)

func WorkspaceAuthenticationMiddleware(intent *string, requiredPermissions *[]entities.MemberPermissions, workspaceSpecific bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		accessToken, _ := ctx.Cookie("workspaceAccessToken")
		appContext, next := middlewares.WorkspaceAuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     savedCtx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, intent, requiredPermissions, accessToken)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
