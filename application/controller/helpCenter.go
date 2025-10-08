package controller

import (
	"context"
	"net/http"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/entities"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
)

func CreateHelpRequest(ctx *interfaces.ApplicationContext[dto.CreateHelpRequestDTO]) {
	// Validate the request payload
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	workspaceID := ctx.GetStringContextData("WorkspaceID")
	memberID := ctx.GetStringContextData("UserID")

	if workspaceID == "" || memberID == "" {
		apperrors.AuthenticationError(ctx.Ctx, "workspace authentication required", ctx.DeviceID)
		return
	}

	// Create the help request entity
	helpRequest := entities.HelpCenter{
		Summary:   ctx.Body.Summary,
		Details:   ctx.Body.Details,
		Workspace: workspaceID,
		Member:    memberID,
		Status:    "open",
	}

	// Save to database
	helpRepo := repository.HelpCenterRepository()
	result, err := helpRepo.CreateOne(context.TODO(), helpRequest)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	// Return success response
	response := dto.HelpCenterResponseDTO{
		ID:        result.ID,
		Summary:   result.Summary,
		Details:   result.Details,
		Workspace: result.Workspace,
		Member:    result.Member,
		Status:    result.Status,
		CreatedAt: result.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: result.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusCreated, "help request created successfully", response, nil, nil, &ctx.DeviceID)
}

func FetchHelpRequests(ctx *interfaces.ApplicationContext[dto.FetchHelpRequestsDTO]) {
	// Validate the request payload
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}

	// Get workspace ID from context (set by workspace auth middleware)
	workspaceID := ctx.GetStringContextData("WorkspaceID")
	if workspaceID == "" {
		apperrors.AuthenticationError(ctx.Ctx, "workspace authentication required", ctx.DeviceID)
		return
	}

	helpRepo := repository.HelpCenterRepository()
	requests, err := helpRepo.FindManyPaginated(map[string]interface{}{
		"workspace": workspaceID,
	}, ctx.Body.Limit, ctx.Body.LastID, 1)
	if err != nil {
		apperrors.UnknownError(ctx.Ctx, err, nil, ctx.DeviceID)
		return
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "help requests fetched successfully", requests, nil, nil, &ctx.DeviceID)
}
