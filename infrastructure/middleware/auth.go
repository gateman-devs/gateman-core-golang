package middlewares

import (
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"github.com/gin-gonic/gin"
)

func AuthenticationMiddleware(business_route bool, restricted bool) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		appContext, next := middlewares.AuthenticationMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:    ctx,
			Keys:   ctx.Keys,
			Header: ctx.Request.Header,
		}, restricted, business_route)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
