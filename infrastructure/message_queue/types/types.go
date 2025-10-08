package mq_types

import "time"

type TaskQueueBroker interface {
	Start()
	Enqueue(task QueueTask)
}

type QueueTask struct {
	Name      Queues
	Payload   []byte
	Priority  TaskPriority
	ProcessIn time.Duration // second
	TimeOut   time.Duration // seconds
	MaxRetry  int
}

type TaskPriority string

const (
	Low    TaskPriority = "low"
	Medium TaskPriority = "medium"
	High   TaskPriority = "high"
)

type BasePayload struct {
	RetryInterval time.Duration
}
