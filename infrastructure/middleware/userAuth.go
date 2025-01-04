package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"authone.usepolymer.co/entities"
	"github.com/gin-gonic/gin"
)

func UserAuthenticationMiddleware(intent string, requiredPermissions *[]entities.MemberPermissions, workspaceSpecific bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		appContext, next := middlewares.UserAuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     savedCtx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: ctx.Request.Header.Get("X-Device-Id"),
		}, intent, requiredPermissions, workspaceSpecific)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
