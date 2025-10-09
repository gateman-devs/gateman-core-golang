package asynq

import (
	"encoding/json"
	"os"
	"time"

	queue_tasks "gateman.io/infrastructure/message_queue/tasks"
	mq_types "gateman.io/infrastructure/message_queue/types"
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
			Password: os.Getenv("REDIS_PASSWORD"),
		},
		asynq.Config{
			Concurrency: 500,
			Queues: map[string]int{
				string(mq_types.High):   7,
				string(mq_types.Medium): 2,
				string(mq_types.Low):    1,
			},
			RetryDelayFunc: func(n int, err error, t *asynq.Task) time.Duration {
				var payload struct{ mq_types.BasePayload }
				if err := json.Unmarshal(t.Payload(), &payload); err != nil {
					return 24 * time.Hour
				}
				return payload.RetryInterval
			},
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(string(queue_tasks.HandleEmailDeliveryTaskName), queue_tasks.HandleEmailDeliveryTask)
	mux.HandleFunc(string(queue_tasks.HandleWorkspaceInviteTaskName), queue_tasks.HandleWorkspaceInviteTask)
	mux.HandleFunc(string(queue_tasks.HandleAppDeletionTaskName), queue_tasks.HandleAppDeletionTask)
	mux.HandleFunc(string(queue_tasks.HandleSubscriptionAutoRenewal), queue_tasks.HandleSubsciptionAutoRenewalTask)

	srv.Run(mux)
}

func (aq *AsynqBroker) Enqueue(task mq_types.QueueTask) {
	if task.TimeOut < 1 {
		task.TimeOut = 60
	}
	task.TimeOut = 60000000000
	aq.Client.Enqueue(asynq.NewTask(string(task.Name), task.Payload),
		asynq.ProcessIn(time.Duration(task.ProcessIn)*time.Second),
		asynq.MaxRetry(task.MaxRetry),
		asynq.ProcessIn(task.ProcessIn),
		asynq.Timeout(task.TimeOut),
		asynq.Queue(string(task.Priority)))
}
