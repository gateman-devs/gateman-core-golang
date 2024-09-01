package controller

import (
	"net/http"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	org_usecases "authone.usepolymer.co/application/usecases/organisation"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func CreateOrganisation(ctx *interfaces.ApplicationContext[dto.CreateOrgDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	err := org_usecases.CreateOrgUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, ctx.DeviceID)
}
