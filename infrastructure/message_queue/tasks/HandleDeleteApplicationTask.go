package queue_tasks

import (
	"context"
	"encoding/json"

	"authone.usepolymer.co/application/repository"
	"authone.usepolymer.co/infrastructure/logger"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
	"github.com/hibiken/asynq"
)

var HandleAppDeletionTaskName mq_types.Queues = "delete_app"

type DeleteAppPayload struct {
	ID          string
	WorkspaceID string
}

func HandleAppDeletionTask(ctx context.Context, t *asynq.Task) error {
	var payload DeleteAppPayload
	err := json.Unmarshal(t.Payload(), &payload)
	if err != nil {
		logger.Error("an error occured while unmarshalling delete app queue payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	appRepo := repository.ApplicationRepo()
	app, _ := appRepo.FindByID(payload.ID)
	if app.DeletedAt == nil {
		logger.Info("app deletion stopped because app has been recovered", logger.LoggerOptions{
			Key:  "payload",
			Data: payload,
		})
		return nil
	}
	appUserRepo := repository.AppUserRepo()
	appRepo.RemoveFromDatabase(context.TODO(), map[string]any{
		"_id":         payload.ID,
		"workspaceID": payload.WorkspaceID,
	})
	appUserRepo.RemoveFromDatabase(context.TODO(), map[string]interface{}{
		"appID":       payload.ID,
		"workspaceID": payload.WorkspaceID,
	})
	return nil
}
