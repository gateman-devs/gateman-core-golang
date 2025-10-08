package queue_tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gateman.io/infrastructure/database/repository/cache"
	"gateman.io/infrastructure/logger"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/messaging/emails"
	"github.com/hibiken/asynq"
)

var HandleEmailDeliveryTaskName mq_types.Queues = "send_email"

type EmailPayload struct {
	To       string
	Subject  string
	Template string
	Opts     map[string]any
	Intent   string
}

func HandleEmailDeliveryTask(ctx context.Context, t *asynq.Task) error {
	var payload EmailPayload
	err := json.Unmarshal(t.Payload(), &payload)
	if err != nil {
		logger.Error("an error occured while unmarshalling email queue payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	success := emails.EmailService.SendEmail(payload.To, payload.Subject, payload.Template, payload.Opts)
	if !success {
		logger.Error("failed to send email", logger.LoggerOptions{
			Key:  "toEmail",
			Data: payload.To,
		}, logger.LoggerOptions{
			Key:  "templateName",
			Data: payload.Template,
		})
		return fmt.Errorf("failed to send email to %s", payload.To)
	}
	if payload.Intent != "" {
		cache.Cache.CreateEntry(fmt.Sprintf("%s-otp-intent", payload.To), payload.Intent, time.Minute*10)
	}
	return nil
}
