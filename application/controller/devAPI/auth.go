package public_controller

import (
	"net/http"

	"gateman.io/application/interfaces"
	application_usecase "gateman.io/application/usecases/application"
	server_response "gateman.io/infrastructure/serverResponse"
)

func APIVerifyAuthToken(ctx *interfaces.ApplicationContext[any]) {
	_, err := application_usecase.VerifyAuthTokenUseCase(ctx.Ctx, ctx.Param["token"].(string))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "token verified", nil, nil, nil)
}
