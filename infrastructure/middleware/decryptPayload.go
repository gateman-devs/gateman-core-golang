package middlewares

import (
	"encoding/hex"
	"io"

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
		decryptedBody := middlewares.DecryptPayloadMiddleware(&interfaces.ApplicationContext[string]{
			Ctx:      ctx,
			Body:     utils.GetStringPointer(string(body)),
			DeviceID: ctx.GetHeader("X-Device-Id"),
		})
		ctx.Set("DecryptedBody", utils.GetStringPointer(hex.EncodeToString(decryptedBody)))
		ctx.Next()
	}
}
