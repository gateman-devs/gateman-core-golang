package messagequeue

import (
	"gateman.io/infrastructure/message_queue/asynq"
	mq_types "gateman.io/infrastructure/message_queue/types"
)

var TaskQueue mq_types.TaskQueueBroker = &asynq.AsynqBroker{}

func StartQueue() {
	TaskQueue.Start()
}
