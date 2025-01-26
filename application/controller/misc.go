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
	var url *string
	var err error
	if ctx.Body.Permission.Read {
		url, err = fileupload.FileUploader.GenerateDownloadURL(ctx.Body.FilePath)
	} else if ctx.Body.Permission.Write {
		url, err = fileupload.FileUploader.GenerateUploadURL(ctx.Body.FilePath)
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
