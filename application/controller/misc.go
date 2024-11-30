package controller

import (
	"fmt"
	"net/http"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	fileupload "authone.usepolymer.co/infrastructure/file_upload"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
)

func GeneratedSignedURL(ctx *interfaces.ApplicationContext[dto.GeneratedSignedURLDTO]) {
	if ctx.Body.AccountImage {
		ctx.Body.FilePath = fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), "accountimage")
	}
	url, err := fileupload.FileUploader.GeneratedSignedURL(ctx.Body.FilePath, ctx.Body.Permission)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "account created", map[string]any{
		"url":      url,
		"filePath": ctx.Body.FilePath,
	}, nil, nil, nil, nil)
}
