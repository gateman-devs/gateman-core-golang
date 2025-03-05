package controller

import (
	"net/http"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	org_usecases "gateman.io/application/usecases/workspace"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
)

func CreateWorkspace(ctx *interfaces.ApplicationContext[dto.CreateWorkspaceDTO]) {
	valiedationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if valiedationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, valiedationErr)
		return
	}
	err := org_usecases.CreateWorkspaceUseCase(ctx.Ctx, ctx.Body, ctx.DeviceID, ctx.DeviceName, ctx.UserAgent, ctx.Param["ip"].(string))
	if err != nil {
		return
	}
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org created", nil, nil, nil, &ctx.DeviceID)
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
	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "org updated", nil, nil, nil, &ctx.DeviceID)
}
