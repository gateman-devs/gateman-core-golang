package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"authone.usepolymer.co/application/utils"
	"authone.usepolymer.co/entities"
	"github.com/gin-gonic/gin"
)

func AuthenticationMiddleware(restricted bool, requiredPermissions *[]entities.MemberPermissions) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appContext, next := middlewares.AuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     ctx.Keys,
			Header:   ctx.Request.Header,
			DeviceID: utils.GetStringPointer(ctx.Request.Header.Get("X-Device-Id")),
		}, restricted, requiredPermissions)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
