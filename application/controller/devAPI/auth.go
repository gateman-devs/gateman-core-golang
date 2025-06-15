package public_controller

import (
	"gateman.io/application/interfaces"
)

func APIVerifyAuthToken(ctx *interfaces.ApplicationContext[any]) {
	//	 err := application_usecase.VerifyAuthTokenUseCase(ctx.Ctx, ctx.Param["token"].(string),ctx.Param["appID"].(string) )
	//		if err != nil {
	//			return
	//		}
	//		server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "token verified", nil, nil, nil)
}
