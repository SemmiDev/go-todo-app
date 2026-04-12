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

	// TaskSendWelcomeEmail is the unique identifier for the welcome email task.
	TaskSendWelcomeEmail = "task:send_welcome_email"

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
	return d.enqueueTask(ctx, TaskSendReminderEmail, payload, QueueDefault)
}

// DistributeTaskSendWelcomeEmail serializes the payload and enqueues a SendWelcomeEmail task 
// into the critical queue with a maximum of 5 retries.
func (d *Distributor) DistributeTaskSendWelcomeEmail(
	ctx context.Context,
	payload *output.TaskPayloadSendWelcomeEmail,
) error {
	return d.enqueueTask(ctx, TaskSendWelcomeEmail, payload, QueueCritical)
}

// enqueueTask is a helper method to serialize the payload and enqueue an Asynq task.
func (d *Distributor) enqueueTask(
	ctx context.Context,
	taskType string,
	payload any,
	queue string,
) error {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	task := asynq.NewTask(taskType, jsonPayload,
		asynq.MaxRetry(5),
		asynq.Timeout(10*time.Second),
		asynq.Queue(queue),
	)

	_, err = d.client.EnqueueContext(ctx, task)
	if err != nil {
		return fmt.Errorf("enqueue task: %w", err)
	}

	return nil
}
