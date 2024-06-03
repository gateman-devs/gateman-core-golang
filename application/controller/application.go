package controller

import (
	"errors"
	"fmt"
	"net/http"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/constants"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	application_usecase "authone.usepolymer.co/application/usecases/application"
	"authone.usepolymer.co/application/utils"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func CreateApplication(ctx *interfaces.ApplicationContext[dto.ApplicationDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
		return
	}
	if len(ctx.Body.RequestedFields) == 0 {
		apperrors.ValidationFailedError(ctx.Ctx, &[]error{errors.New("requestedFields cannot be empty")}, ctx.DeviceID)
		return
	}
	for _, field := range *ctx.Body.RequiredVerifications {
		if !utils.HasItemString(&constants.AVAILABLE_REQUIRED_DATA_POINTS, field) {
			apperrors.ValidationFailedError(ctx.Ctx, &[]error{fmt.Errorf("%s is not allowed in requested field", field)}, ctx.DeviceID)
			return
		}
	}
	if ctx.Body.LocaleRestriction != nil {
		for _, r := range *ctx.Body.LocaleRestriction {
			valiedationErr := validator.ValidatorInstance.ValidateStruct(r)
			if valiedationErr != nil {
				apperrors.ValidationFailedError(ctx.Ctx, valiedationErr, ctx.DeviceID)
				return
			}
		}
	}
	app, apiKey := application_usecase.CreateApplicationUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("OrgID"), ctx.GetStringContextData("Email"))
	if app == nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", map[string]any{
		"app":    app,
		"apiKey": apiKey,
	}, nil, nil, ctx.DeviceID)
}

func FetchAppCreationConfigInfo(ctx *interfaces.ApplicationContext[any]) {
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "required fields", map[string]any{
		"requiredFields":  constants.AVAILABLE_REQUIRED_DATA_POINTS,
		"requestedFields": []string{},
	}, nil, nil, ctx.DeviceID)
}
