package controller

import (
	"net/http"

	apperrors "authone.usepolymer.co/application/appErrors"
	"authone.usepolymer.co/application/controller/dto"
	"authone.usepolymer.co/application/interfaces"
	"authone.usepolymer.co/application/repository"
	org_usecases "authone.usepolymer.co/application/usecases/organisation"
	"authone.usepolymer.co/infrastructure/logger"
	server_response "authone.usepolymer.co/infrastructure/serverResponse"
	"authone.usepolymer.co/infrastructure/validator"
)

func CreateOrganisation(ctx *interfaces.ApplicationContext[dto.CreateOrgDTO]) {
	err := org_usecases.CreateOrgUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.UserAgent, ctx.GetStringContextData("UserID"), ctx.GetStringContextData("Email"))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, nil, nil)
}

func FetchWorkspaces(ctx *interfaces.ApplicationContext[any]) {
	WorkspaceMemberRepo := repository.WorkspaceMemberRepo()
	workspaces, err := WorkspaceMemberRepo.FindMany(map[string]interface{}{
		"userID": ctx.GetStringContextData("UserID"),
	})
	if err != nil {
		logger.Error("error fetching users orgs", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.UnknownError(ctx.Ctx, err)
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "workspaces fetched", workspaces, nil, nil, nil, nil)
}

func UpdateOrgDetails(ctx *interfaces.ApplicationContext[dto.UpdateOrgDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	workspaceRepo := repository.WorkspaceRepository()
	workspaceRepo.UpdatePartialByFilter(map[string]interface{}{
		"_id": ctx.GetStringContextData("WorkspaceID"),
	}, ctx.Body)
	if ctx.Body.WorkspaceName != nil {
		workspaceMemberRepo := repository.WorkspaceMemberRepo()
		workspaceMemberRepo.UpdatePartialByFilter(map[string]interface{}{
			"workspaceID": ctx.GetStringContextData("WorkspaceID"),
		}, map[string]any{
			"workspaceName": ctx.Body.WorkspaceName,
		})
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org updated", nil, nil, nil, nil, nil)
}
