package queue_tasks

import (
	"context"
	"encoding/json"

	"authone.usepolymer.co/infrastructure/logger"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
	"authone.usepolymer.co/infrastructure/messaging/sms"
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
