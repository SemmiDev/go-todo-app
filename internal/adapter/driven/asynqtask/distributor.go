// Package asynqtask provides the Asynq-based implementation for distributing 
// background tasks to Redis-backed queues.
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
	// TaskSendReminderEmail is the unique identifier for the reminder email task.
	TaskSendReminderEmail = "task:send_reminder_email"

	// QueueCritical handles high-priority tasks.
	QueueCritical = "critical"
	// QueueDefault handles standard-priority tasks.
	QueueDefault = "default"
)

// Distributor implements output.TaskDistributor using Asynq and Redis.
// It is responsible for serializing task payloads and enqueuing them.
type Distributor struct {
	client *asynq.Client
}

// NewDistributor constructs a new Asynq task distributor with the given Redis options.
func NewDistributor(redisOpt asynq.RedisClientOpt) *Distributor {
	client := asynq.NewClient(redisOpt)
	return &Distributor{
		client: client,
	}
}

// DistributeTaskSendReminderEmail serializes the payload and enqueues a SendReminderEmail task 
// into the default queue with a maximum of 5 retries.
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
