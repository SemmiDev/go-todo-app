// Package reminder implements the application service for sending todo reminder emails.
// It is driven by a scheduled job and depends on output ports to find and enqueue reminder tasks.
package reminder

import (
	"context"
	"log/slog"
	"time"

	"github.com/semmidev/go-todo-app/internal/port/output"
)

// Service orchestrates the workflow of finding due todos and enqueuing reminder tasks.
type Service struct {
	todoRepo        output.TodoRepository
	taskDistributor output.TaskDistributor
	logger          *slog.Logger
	window          time.Duration // how far ahead to look (default 24h)
}

// Config holds optional configuration overrides for the reminder service.
type Config struct {
	Window time.Duration // default 24h
	AppURL string        // default "http://localhost:8080"
}

// NewService creates a new reminder service with the provided dependencies and configuration.
func NewService(
	todoRepo output.TodoRepository,
	taskDistributor output.TaskDistributor,
	logger *slog.Logger,
	cfg Config,
) *Service {
	if cfg.Window == 0 {
		cfg.Window = 24 * time.Hour
	}
	return &Service{
		todoRepo:        todoRepo,
		taskDistributor: taskDistributor,
		logger:          logger,
		window:          cfg.Window,
	}
}

// SendDueSoonReminders is the job entry-point called by the scheduler to process pending reminders.
func (s *Service) SendDueSoonReminders(ctx context.Context) error {
	start := time.Now()

	todos, err := s.todoRepo.FindDueSoon(ctx, s.window)
	if err != nil {
		s.logger.ErrorContext(ctx, "reminder: query failed", slog.Any("error", err))
		return err
	}

	if len(todos) == 0 {
		return nil
	}

	enqueued, failed := 0, 0
	for _, t := range todos {
		triggered := t.TriggeredReminders()
		triggeredMap := make(map[string]bool)
		for _, offset := range triggered {
			triggeredMap[offset] = true
		}

		for _, offset := range t.Reminders() {
			// Skip if already sent
			if triggeredMap[offset] {
				continue
			}

			// Parse offset (e.g., "1h", "15m")
			d, err := time.ParseDuration(offset)
			if err != nil {
				s.logger.WarnContext(ctx, "reminder: invalid offset", slog.String("offset", offset))
				continue
			}

			// Check if it's time to send (due_date - offset <= now)
			if t.DueDate().Add(-d).Before(time.Now()) {
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
