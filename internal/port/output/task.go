// Package output defines the outbound ports (interfaces) of the application.
// These interfaces are implemented by the driven adapters (infrastructure) and called by the application layer.
package output

import (
	"context"

	"github.com/google/uuid"
)

// TaskPayloadSendReminderEmail contains the data needed to process a reminder email task.
type TaskPayloadSendReminderEmail struct {
	// TodoID is the unique identifier of the todo for which to send a reminder.
	TodoID uuid.UUID `json:"todo_id"`
}

// TaskPayloadSendWelcomeEmail contains the data needed to process a welcome email task.
type TaskPayloadSendWelcomeEmail struct {
	// UserID is the unique identifier of the user who registered.
	UserID uuid.UUID `json:"user_id"`
}

// TaskDistributor is the driven port for enqueuing background tasks into a message queue.
type TaskDistributor interface {
	// DistributeTaskSendReminderEmail schedules a task to send a reminder email.
	DistributeTaskSendReminderEmail(ctx context.Context, payload *TaskPayloadSendReminderEmail) error

	// DistributeTaskSendWelcomeEmail schedules a task to send a welcome email.
	DistributeTaskSendWelcomeEmail(ctx context.Context, payload *TaskPayloadSendWelcomeEmail) error
}
