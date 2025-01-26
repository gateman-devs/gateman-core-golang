package public_controller

import (
	"net/http"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	application_usecase "authone.usepolymer.co/application/usecases/application"
	"authone.usepolymer.co/infrastructure/logger"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
)

func APIFetchAppDetails(ctx *interfaces.ApplicationContext[any]) {
	app, err := application_usecase.FetchAppUseCase(ctx.Ctx, ctx.Param["id"].(string), ctx.DeviceID, ctx.Keys["ip"].(string))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app fetched", app, nil, nil, nil, nil)
}

func APIFetchAppUser(ctx *interfaces.ApplicationContext[any]) {
	appUserRepo := repository.AppUserRepo()
	appUser, err := appUserRepo.FindOneByFilter(map[string]interface{}{
		"_id":         ctx.GetStringContextData("UserID"),
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
	})
	if err != nil {
		logger.Error("an error occured while fetching app user", logger.LoggerOptions{
			Key:  "userID",
			Data: ctx.GetStringContextData("UserID"),
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "app user fetched", appUser, nil, nil, nil, nil)
}

func APIFetchAppUsers(ctx *interfaces.ApplicationContext[dto.FetchAppUsersDTO]) {
	filter := map[string]interface{}{
		"appID":       ctx.Body.AppID,
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
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
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users fetched", users, nil, nil, nil, nil)
}

func APIBlockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
	}, map[string]any{
		"blocked": true,
	})
	if err != nil {
		logger.Error("an error occured while blocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users blocked", nil, nil, nil, nil, nil)
}

func APIUnblockAccounts(ctx *interfaces.ApplicationContext[dto.BlockAccountsDTO]) {
	appUserRepo := repository.AppUserRepo()
	_, err := appUserRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": map[string]any{
			"$in": ctx.Body.IDs,
		},
		"appID":       ctx.GetStringParameter("id"),
		"workspaceID": ctx.GetHeader("X-Workspace-Id"),
	}, map[string]any{
		"blocked": false,
	})
	if err != nil {
		logger.Error("an error occured while unblocking users", logger.LoggerOptions{
			Key:  "ids",
			Data: ctx.Body.IDs,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "users unblocked", nil, nil, nil, nil, nil)
}
