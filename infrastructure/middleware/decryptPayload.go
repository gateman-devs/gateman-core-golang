package middlewares

import (
	"io"
	"strings"

	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/middlewares"
	"authone.usepolymer.co/application/utils"
	"github.com/gin-gonic/gin"
)

func DecryptPayloadMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		body, err := io.ReadAll(ctx.Request.Body)
		if err != nil {
			ctx.Next()
			ctx.Set("AppContext", &interfaces.ApplicationContext[any]{
				Ctx: ctx,
			})
			return
		}
		bodyString := strings.Split(string(body), ".")
		payload := bodyString[0]
		nonce := bodyString[1]
		decryptedBody := middlewares.DecryptPayloadMiddleware(&interfaces.ApplicationContext[string]{
			Ctx:      ctx,
			Body:     utils.GetStringPointer(payload),
			Nonce:    utils.GetStringPointer(nonce),
			DeviceID: utils.GetStringPointer(ctx.GetHeader("X-Device-Id")),
		})
		ctx.Set("DecryptedBody", *decryptedBody)
		ctx.Next()
	}
}
