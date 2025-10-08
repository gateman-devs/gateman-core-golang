package controller

import (
	"net/http"
	"time"

	apperrors "gateman.io/application/appErrors"
	"gateman.io/application/controller/dto"
	"gateman.io/application/interfaces"
	"gateman.io/application/repository"
	"gateman.io/infrastructure/logger"
	server_response "gateman.io/infrastructure/serverResponse"
	"gateman.io/infrastructure/validator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FetchActivityLogs(ctx *interfaces.ApplicationContext[dto.FetchActivityLogsDTO]) {
	validationErr := validator.ValidatorInstance.ValidateStruct(ctx.Body)
	if validationErr != nil {
		apperrors.ValidationFailedError(ctx.Ctx, validationErr, ctx.DeviceID)
		return
	}
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindOneByFilter(map[string]interface{}{
		"_id":         ctx.Body.AppID,
		"workspaceID": ctx.Keys["WorkspaceID"],
	})
	if app == nil {
		apperrors.ClientError(ctx.Ctx, "application not found", nil, nil, ctx.DeviceID)
		return
	}
	if app.WorkspaceID != ctx.Keys["WorkspaceID"] {
		apperrors.ClientError(ctx.Ctx, "application not found", nil, nil, ctx.DeviceID)
		return
	}

	// Build filter
	filter := map[string]interface{}{
		"appID": app.AppID,
	}

	// Filter by IP address
	if ctx.Body.IPAddress != nil && *ctx.Body.IPAddress != "" {
		filter["ipAddress"] = *ctx.Body.IPAddress
	}

	// Filter by method
	if ctx.Body.Method != nil && *ctx.Body.Method != "" {
		filter["method"] = *ctx.Body.Method
	}

	// Filter by URL with fuzzy search (matches any part of the URL)
	if ctx.Body.URL != nil && *ctx.Body.URL != "" {
		filter["url"] = bson.M{"$regex": *ctx.Body.URL, "$options": "i"}
	}

	// Filter by time range
	if ctx.Body.StartTime != nil || ctx.Body.EndTime != nil {
		timeFilter := bson.M{}

		if ctx.Body.StartTime != nil && *ctx.Body.StartTime != "" {
			startTime, err := time.Parse(time.RFC3339, *ctx.Body.StartTime)
			if err != nil {
				apperrors.ClientError(ctx.Ctx, "invalid startTime format, use RFC3339", nil, nil, ctx.DeviceID)
				return
			}
			timeFilter["$gte"] = startTime
		}

		if ctx.Body.EndTime != nil && *ctx.Body.EndTime != "" {
			endTime, err := time.Parse(time.RFC3339, *ctx.Body.EndTime)
			if err != nil {
				apperrors.ClientError(ctx.Ctx, "invalid endTime format, use RFC3339", nil, nil, ctx.DeviceID)
				return
			}
			timeFilter["$lte"] = endTime
		}

		if len(timeFilter) > 0 {
			filter["timestamp"] = timeFilter
		}
	}

	// Set default pagination values
	pageSize := int64(50)
	if ctx.Body.PageSize != nil && *ctx.Body.PageSize > 0 {
		pageSize = *ctx.Body.PageSize
		if pageSize > 100 {
			pageSize = 100 // Max limit
		}
	}

	sortOrder := -1 // Default to descending (newest first)
	if ctx.Body.SortOrder != nil && (*ctx.Body.SortOrder == 1 || *ctx.Body.SortOrder == -1) {
		sortOrder = *ctx.Body.SortOrder
	}

	// Fetch activity logs with pagination
	logs, err := repository.RequestActivityLogRepo().FindManyPaginated(
		filter,
		pageSize,
		ctx.Body.LastID,
		sortOrder,
		options.Find().SetSort(bson.M{"timestamp": sortOrder}),
	)

	if err != nil {
		logger.Error("error fetching activity logs", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		apperrors.FatalServerError(ctx.Ctx, err, ctx.DeviceID)
		return
	}

	// Get total count for the filter
	totalCount, err := repository.RequestActivityLogRepo().CountDocs(filter)
	if err != nil {
		logger.Error("error counting activity logs", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		totalCount = 0
	}

	server_response.Responder.Respond(ctx.Ctx, http.StatusOK, "activity logs fetched successfully", map[string]any{
		"logs":       logs,
		"totalCount": totalCount,
		"pageSize":   pageSize,
	}, nil, nil, &ctx.DeviceID)
}
