package asynqtask

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

const (
	TaskSendReminderEmail = "task:send_reminder_email"
	QueueCritical         = "critical"
	QueueDefault          = "default"
)

// Distributor implements output.TaskDistributor using Asynq (Redis).
type Distributor struct {
	client *asynq.Client
}

// NewDistributor constructs a new Asynq task distributor.
func NewDistributor(redisOpt asynq.RedisClientOpt) *Distributor {
	client := asynq.NewClient(redisOpt)
	return &Distributor{
		client: client,
	}
}

// DistributeTaskSendReminderEmail enqueues a SendReminderEmail task.
func (d *Distributor) DistributeTaskSendReminderEmail(
	ctx context.Context,
	payload *output.TaskPayloadSendReminderEmail,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	task := asynq.NewTask(TaskSendReminderEmail, jsonPayload,
		asynq.MaxRetry(5),
		asynq.Timeout(10*time.Second),
		asynq.Queue(QueueDefault),
	)

	_, err = d.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("enqueue task: %w", err)
	}

	return nil
}
