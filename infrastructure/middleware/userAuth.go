package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"github.com/gin-gonic/gin"
)

func UserAuthenticationMiddleware(intent string, requiredPermissions *[]entities.MemberPermissions, workspaceSpecific bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appContext, next := middlewares.UserAuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: utils.GetStringPointer(ctx.Request.Header.Get("X-Device-Id")),
			Param: map[string]any{
				"workspaceID": ctx.Param("workspaceID"),
			},
		}, intent, requiredPermissions, workspaceSpecific)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
