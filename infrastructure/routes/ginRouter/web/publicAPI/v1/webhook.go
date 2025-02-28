package v1

import (
	"fmt"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller"
	"gateman.io/application/interfaces"
	"github.com/gin-gonic/gin"
)

func WebhookRouter(router *gin.RouterGroup) {
	webhookRouter := router.Group("/webhooks")
	{
		webhookRouter.POST("/paystack", func(ctx *gin.Context) {
			body, err := ctx.GetRawData()
			if err != nil {
				apperrors.ErrorProcessingPayload(ctx)
				return
			}
			fmt.Println(body)
			controller.ProcessPaystackWebhook(&interfaces.ApplicationContext[[]byte]{
				Ctx:    ctx,
				Body:   &body,
				Header: ctx.Request.Header,
			})
		})
	}
}
