package middlewares

import (
	"encoding/hex"
	"io"

	"gateman.io/application/interfaces"
	"gateman.io/application/middlewares"
	"gateman.io/application/utils"
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
		decryptedBody := middlewares.DecryptPayloadMiddleware(&interfaces.ApplicationContext[string]{
			Ctx:      ctx,
			Body:     utils.GetStringPointer(string(body)),
			DeviceID: ctx.GetHeader("X-Device-Id"),
		})
		ctx.Set("DecryptedBody", utils.GetStringPointer(hex.EncodeToString(decryptedBody)))
		ctx.Next()
	}
}
