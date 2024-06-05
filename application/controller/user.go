package controller

import (
	"net/http"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	user_usecases "authone.usepolymer.co/application/usecases/user"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func CreateUser(ctx *interfaces.ApplicationContext[dto.CreateUserDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	err := user_usecases.CreateUserUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent)
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "account created", nil, nil, nil, ctx.DeviceID)
}
