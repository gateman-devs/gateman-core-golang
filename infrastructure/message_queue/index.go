package messagequeue

import (
	"authone.usepolymer.co/infrastructure/message_queue/asynq"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
)

var TaskQueue mq_types.TaskQueueBroker = &asynq.AsynqBroker{}

func StartQueue() {
	TaskQueue.Start()
}
