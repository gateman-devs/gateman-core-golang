package middlewares

import (
	"fmt"

	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"github.com/gin-gonic/gin"
)

func OTPTokenMiddleware(intent string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		savedCtx := (ctx.MustGet("AppContext")).(*interfaces.ApplicationContext[any])
		accessToken, _ := ctx.Cookie("otpAccessToken")
		fmt.Println(ctx.Request.Cookies())
		fmt.Println("the access", accessToken)
		fmt.Println(accessToken)
		appContext, next := middlewares.OTPTokenMiddleware(&interfaces.ApplicationContext[any]{
			Ctx:      ctx,
			Keys:     savedCtx.Keys,
			DeviceID: savedCtx.DeviceID,
			Header:   ctx.Request.Header,
		}, ctx.ClientIP(), intent, accessToken)
		if next {
			ctx.Set("AppContext", appContext)
			ctx.Next()
		}
	}
}
