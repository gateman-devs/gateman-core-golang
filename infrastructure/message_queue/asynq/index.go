package asynq

import (
	"os"
	"time"

	queue_tasks "authone.usepolymer.co/infrastructure/message_queue/tasks"
	mq_types "authone.usepolymer.co/infrastructure/message_queue/types"
	"github.com/hibiken/asynq"
)

type EmailDeliveryPayload struct {
	UserID     int
	TemplateID string
}

type AsynqBroker struct {
	Client *asynq.Client
}

func (aq *AsynqBroker) Start() {
	redisConnOpt := asynq.RedisClientOpt{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
	}

	aq.Client = asynq.NewClient(redisConnOpt)

	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     os.Getenv("REDIS_ADDR"),
			Password: os.Getenv("REDIS_PASSWORD")},
		asynq.Config{
			Concurrency: 500,
			Queues: map[string]int{
				string(mq_types.High):   7,
				string(mq_types.Medium): 2,
				string(mq_types.Low):    1,
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(string(queue_tasks.HandleEmailDeliveryTaskName), queue_tasks.HandleEmailDeliveryTask)
	mux.HandleFunc(string(queue_tasks.HandleWorkspaceInviteTaskName), queue_tasks.HandleWorkspaceInviteTask)

	srv.Run(mux)
}

func (aq *AsynqBroker) Enqueue(task mq_types.QueueTask) {
	if task.TimeOut == 0 {
		task.TimeOut = 60
	}
	aq.Client.Enqueue(asynq.NewTask(string(task.Name), task.Payload),
		asynq.ProcessIn(time.Duration(task.ProcessIn)*time.Second),
		asynq.MaxRetry(10),
		asynq.Timeout(time.Second*time.Duration(task.TimeOut)),
		asynq.Queue(string(task.Priority)))
}
