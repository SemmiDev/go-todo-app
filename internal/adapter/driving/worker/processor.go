// Package worker implements background task processing logic.
// It acts as a driving adapter that consumes tasks from Redis-backed Asynq queues.
package worker

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// TaskProcessor defines the interface for an async worker that manages
// life-cycles and handler registrations for background jobs.
type TaskProcessor interface {
	// Start begins non-blocking consumption of tasks.
	Start() error
	// Shutdown gracefully halts task processing.
	Shutdown()
	// ProcessTaskSendReminderEmail decodes the payload and sends the notification email.
	ProcessTaskSendReminderEmail(ctx context.Context, task *asynq.Task) error
	// ProcessTaskSendWelcomeEmail decodes the payload and sends the welcome email.
	ProcessTaskSendWelcomeEmail(ctx context.Context, task *asynq.Task) error
}

// RedisTaskProcessor implements TaskProcessor using an Asynq server.
// It coordinates data repository access and external service interactions.
type RedisTaskProcessor struct {
	server      *asynq.Server
	todoRepo    output.TodoRepository
	userRepo    output.UserRepository
	emailSender output.EmailSender
	logger      *slog.Logger
	appURL      string
}

// NewRedisTaskProcessor initializes an Asynq server with standard and critical
// priority levels and global concurrency settings.
func NewRedisTaskProcessor(
	redisOpt asynq.RedisClientOpt,
	todoRepo output.TodoRepository,
	userRepo output.UserRepository,
	emailSender output.EmailSender,
	logger *slog.Logger,
	appURL string,
) TaskProcessor {
	// asynq.Config handles worker scaling and behavior
	server := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Queues: map[string]int{
				"critical": 10,
				"default":  5,
			},
			Concurrency: 10,
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				logger.ErrorContext(ctx, "process task failed",
					slog.Any("error", err),
					slog.String("type", task.Type()),
					slog.String("payload", string(task.Payload())), // careful: payload could be big
				)
			}),
			// Note: Asynq Logger interface is slightly different, but we could adapt slog.
			// For simplicity we use ErrorHandler for the errors we care about most.
		},
	)

	return &RedisTaskProcessor{
		server:      server,
		todoRepo:    todoRepo,
		userRepo:    userRepo,
		emailSender: emailSender,
		logger:      logger,
		appURL:      appURL,
	}
}

// Start registers task handlers and blocks while listening for new work.
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	// Register specific string queue keys to Go functions
	mux.HandleFunc("task:send_reminder_email", processor.ProcessTaskSendReminderEmail)
	mux.HandleFunc("task:send_welcome_email", processor.ProcessTaskSendWelcomeEmail)

	return processor.server.Start(mux)
}

// Shutdown ensures all in-flight tasks are completed before exiting.
func (processor *RedisTaskProcessor) Shutdown() {
	processor.server.Shutdown()
}
