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
		s.logger.InfoContext(ctx, "reminder: no todos due soon", slog.Duration("window", s.window))
		return nil
	}

	enqueued, failed := 0, 0
	for _, t := range todos {
		payload := &output.TaskPayloadSendReminderEmail{
			TodoID: t.ID(),
		}

		if err := s.taskDistributor.DistributeTaskSendReminderEmail(ctx, payload); err != nil {
			s.logger.ErrorContext(ctx, "reminder: enqueue failed",
				slog.String("todo_id", t.ID().String()),
				slog.Any("error", err),
			)
			failed++
			continue
		}

		if err := s.todoRepo.MarkReminderSent(ctx, t.ID()); err != nil {
			s.logger.WarnContext(ctx, "reminder: mark sent failed",
				slog.String("todo_id", t.ID().String()),
				slog.Any("error", err),
			)
		}
		enqueued++
	}

	// Wide-event style: one log line for the entire job run.
	s.logger.InfoContext(ctx, "reminder: job complete",
		slog.Int("total", len(todos)),
		slog.Int("enqueued", enqueued),
		slog.Int("failed", failed),
		slog.Duration("duration_ms", time.Since(start)),
		slog.Duration("window", s.window),
	)
	return nil
}
