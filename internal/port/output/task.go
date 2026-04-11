package output

import (
	"context"

	"github.com/google/uuid"
)

// TaskPayloadSendReminderEmail is the data payload sent to the queue
// when a reminder needs to be sent for a specific todo.
type TaskPayloadSendReminderEmail struct {
	TodoID uuid.UUID `json:"todo_id"`
}

// TaskDistributor is the output port for enqueuing background tasks.
type TaskDistributor interface {
	DistributeTaskSendReminderEmail(ctx context.Context, payload *TaskPayloadSendReminderEmail) error
}
