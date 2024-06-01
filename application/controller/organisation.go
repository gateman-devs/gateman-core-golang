package controller

import (
	"net/http"

	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	org_usecases "authone.usepolymer.co/application/usecases/organisation"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
)

func CreateOrganisation(ctx *interfaces.ApplicationContext[dto.CreateOrgDTO]) {
	// valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	// if valiedationErr != nil {
	// 	apperrors.ValidationFailedError(ctx, valiedationErr, ctx.DeviceID)
	// 	return
	// }
	err := org_usecases.CreateOrgUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.Nonce)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, ctx.Nonce, ctx.DeviceID)
}
