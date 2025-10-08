package public_controller

import (
	"net/http"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/devAPI/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	fileupload "gateman.io/infrastructure/file_upload"
	"gateman.io/infrastructure/file_upload/types"
	"gateman.io/infrastructure/logger"
	server_response "gateman.io/infrastructure/serverResponse"
)

func APIFetchAppDetails(ctx *interfaces.ApplicationContext[dto.FetchAppDTO]) {
	appRepo := repository.ApplicationRepo()
	app, err := appRepo.FindOneByFilter(map[string]interface{}{
		"appID": ctx.GetStringContextData("AppID"),
	})

	if err != nil {
		logger.Error("an error occured while fetching an app on APIFetchAppDetails", logger.LoggerOptions{
			Key:  "err",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	if ctx.Body.GenerateImgURL {
		appImg, _ := fileupload.FileUploader.GeneratedSignedURL(app.AppImg, types.SignedURLPermission{
			Read: true,
		}, time.Hour*5)
		app.AppImg = *appImg
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app fetched", app, nil, nil, &ctx.DeviceID)
}

func APIFetchAppUser(ctx *interfaces.ApplicationContext[any]) {
	appUserRepo := repository.AppUserRepo()
	appUser, err := appUserRepo.FindOneByFilter(map[string]interface{}{
		"_id":         ctx.GetStringContextData("UserID"),
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	})
	if err != nil {
		logger.Error("an error occured while fetching app user", logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app user fetched", appUser, nil, nil, &ctx.DeviceID)
}

func APIFetchAppUsers(ctx *interfaces.ApplicationContext[dto.FetchAppUsersDTO]) {
	filter := map[string]interface{}{
		"appID":       ctx.Body.AppID,
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}
	if ctx.Body.Blocked != nil {
		filter["blocked"] = ctx.Body.Blocked
	}
	if ctx.Body.Deleted != nil {
		filter["deletedAt"] = map[string]any{"$ne": nil}
	}
	appUserRepo := repository.AppUserRepo()
	users, err := appUserRepo.FindManyPaginated(filter, ctx.Body.PageSize, ctx.Body.LastID, int(ctx.Body.Sort))
	if err != nil {
		logger.Error("an error occured while fetching apps users", logger.LoggerOptions{
			Key:  "appID",
			Data: ctx.Body.AppID,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users fetched", users, nil, nil, &ctx.DeviceID)
}

func APIBlockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}, map[string]any{
		"blocked": true,
	})
	if err != nil {
		logger.Error("an error occured while blocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users blocked", nil, nil, nil, &ctx.DeviceID)
}

func APIUnblockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetStringContextData("WorkspaceID"),
	}, map[string]any{
		"blocked": false,
	})
	if err != nil {
		logger.Error("an error occured while unblocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users unblocked", nil, nil, nil, &ctx.DeviceID)
}
