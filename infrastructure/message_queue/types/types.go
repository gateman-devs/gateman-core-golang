package mq_types

type TaskQueueBroker interface {
	Start()
	Enqueue(task QueueTask)
}

type QueueTask struct {
	Name      Queues
	Payload   []byte
	Priority  TaskPriority
	ProcessIn uint // minute
	TimeOut   uint // seconds
}

type TaskPriority string

const (
	Low    TaskPriority = "low"
	Medium TaskPriority = "medium"
	High   TaskPriority = "high"
)

