package controller

import (
	"fmt"
	"net/http"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
)

func GeneratedSignedURL(ctx *interfaces.ApplicationContext[dto.GeneratedSignedURLDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	if ctx.Body.AccountImage {
		ctx.Body.FilePath = fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage")
	}
	var url *string
	var err error
	if ctx.Body.Permission.Read {
		url, err = fileupload.FileUploader.GeneratedSignedURL(ctx.Body.FilePath, types.SignedURLPermission{
			Read: true,
		})
	} else if ctx.Body.Permission.Write {
		url, err = fileupload.FileUploader.GeneratedSignedURL(ctx.Body.FilePath, types.SignedURLPermission{
			Write: true,
		})
	} else if ctx.Body.Permission.Delete {
	} else {
		apperrors.ClientError(ctx.Ctx, "invalid request", nil, nil)
		return
	}
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "account created", map[string]any{
		"url":      url,
		"filePath": ctx.Body.FilePath,
	}, nil, nil, nil, nil)
}
