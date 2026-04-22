// Package reminder implements the application service for sending todo reminder emails.
// It is driven by a scheduled job and depends on output ports to find and enqueue reminder tasks.
package reminder

import (
	"context"
	"log/slog"
	"time"

	"github.com/semmidev/todo-app/internal/port/output"
)

// Service orchestrates the workflow of finding due todos and enqueuing reminder tasks.
type Service struct {
	todoRepo        output.TodoRepository
	taskDistributor output.TaskDistributor
	logger          *slog.Logger
}

// Config holds optional configuration overrides for the reminder service.
type Config struct {
	AppURL string // default "http://localhost:8080"
}

// NewService creates a new reminder service with the provided dependencies and configuration.
func NewService(
	todoRepo output.TodoRepository,
	taskDistributor output.TaskDistributor,
	logger *slog.Logger,
	cfg Config,
) *Service {
	return &Service{
		todoRepo:        todoRepo,
		taskDistributor: taskDistributor,
		logger:          logger,
	}
}

// SendDueSoonReminders is the job entry-point called by the scheduler to process pending reminders.
func (s *Service) SendDueSoonReminders(ctx context.Context) error {
	start := time.Now()

	todos, err := s.todoRepo.FindDueSoon(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "reminder: query failed", slog.Any("error", err))
		return err
	}

	if len(todos) == 0 {
		return nil
	}

	enqueued, failed := 0, 0
	for _, t := range todos {
		dueOffsets := t.GetDueReminders(start)
		for _, offset := range dueOffsets {
			payload := &output.TaskPayloadSendReminderEmail{
				TodoID: t.ID(),
			}

			if err := s.taskDistributor.DistributeTaskSendReminderEmail(ctx, payload); err != nil {
				s.logger.ErrorContext(ctx, "reminder: enqueue failed",
					slog.String("todo_id", t.ID().String()),
					slog.String("offset", offset),
					slog.Any("error", err),
				)
				failed++
				continue
			}

			if err := s.todoRepo.MarkReminderTriggered(ctx, t.ID(), offset); err != nil {
				s.logger.WarnContext(ctx, "reminder: mark triggered failed",
					slog.String("todo_id", t.ID().String()),
					slog.String("offset", offset),
					slog.Any("error", err),
				)
			}
			enqueued++
		}
	}

	if enqueued > 0 || failed > 0 {
		// Wide-event style: one log line for the entire job run.
		s.logger.InfoContext(ctx, "reminder: job complete",
			slog.Int("enqueued", enqueued),
			slog.Int("failed", failed),
			slog.Duration("duration_ms", time.Since(start)),
		)
	}
	return nil
}
