package routev1

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"authone.usepolymer.co/application/controller"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"github.com/gin-gonic/gin"
)

func AuthRouter(router *gin.RouterGroup) {
	authRouter := router.Group("/auth")
	{
		authRouter.POST("/org/create", func(ctx *gin.Context) {
			appContext := ctx.MustGet("AppContext").(*interfaces.ApplicationContext[any])
			decryptedBody := ctx.MustGet("DecryptedBody").(string)
			fmt.Println(decryptedBody)
			fmt.Println(decryptedBody)
			fmt.Println(decryptedBody)
			var body dto.CreateOrgDTO
			bodyByte, _ := hex.DecodeString(decryptedBody)
			json.Unmarshal(bodyByte, &body)
			fmt.Println(body)
			// if err != nil {
			// 	apperrors.ErrorProcessingPayload(ctx, appContext.DeviceID)
			// 	return
			// }
			controller.CreateOrganisation(&interfaces.ApplicationContext[dto.CreateOrgDTO]{
				Ctx:        ctx,
				Body:       &body,
				DeviceID:   appContext.DeviceID,
				UserAgent:  appContext.UserAgent,
				AppVersion: appContext.AppVersion,
			})
		})
	}
}
