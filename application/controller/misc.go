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
	var fileName string
	if ctx.Body.AccountImage {
		fileName = "accountimage"
	} else {
		fileName = *ctx.DeviceID
	}
	filePath := fmt.Sprintf("%s/%s", ctx.GetStringContextData("UserID"), fileName)
	url, err := fileupload.FileUploader.GeneratedSignedURL(filePath, ctx.Body.Permission)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "account created", map[string]any{
		"url":      url,
		"filePath": filePath,
	}, nil, nil, ctx.DeviceID)
}
