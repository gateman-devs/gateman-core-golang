package queue_tasks

import (
	"context"
	"encoding/json"

	"gateman.io/infrastructure/logger"
	mq_types "gateman.io/infrastructure/message_queue/types"
	"gateman.io/infrastructure/messaging/sms"
	"github.com/hibiken/asynq"
)

var HandleOTPDeliveryTaskName mq_types.Queues = "send_sms"

type SendOTPPayload struct {
	To       string
	Msg      string
	Whatsapp bool
}

func HandleOTPDeliveryTask(ctx context.Context, t *asynq.Task) error {
	var payload SendOTPPayload
	err := json.Unmarshal(t.Payload(), &payload)
	if err != nil {
		logger.Error("an error occured while unmarshalling otp queue payload", logger.LoggerOptions{
			Key:  "error",
			Data: err,
		})
		return err
	}
	sms.SMSService.SendOTP(payload.To, payload.Whatsapp, &payload.Msg)
	return nil
}
