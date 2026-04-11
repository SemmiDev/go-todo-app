package worker

import (
	"context"
	"log/slog"

	"github.com/hibiken/asynq"
	"github.com/semmidev/go-todo-app/internal/port/output"
)

// TaskProcessor defines the driving adapter interface for processing async queues.
type TaskProcessor interface {
	Start() error
	Shutdown()
	ProcessTaskSendReminderEmail(ctx context.Context, task *asynq.Task) error
}

// RedisTaskProcessor implements TaskProcessor using Asynq.
type RedisTaskProcessor struct {
	server      *asynq.Server
	todoRepo    output.TodoRepository
	userRepo    output.UserRepository
	emailSender output.EmailSender
	logger      *slog.Logger
	appURL      string
}

// NewRedisTaskProcessor constructs the processor and configures concurrency & error handling.
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

// Start registers the Mux and blocks consuming tasks.
func (processor *RedisTaskProcessor) Start() error {
	mux := asynq.NewServeMux()

	// Register specific string queue keys to Go functions
	mux.HandleFunc("task:send_reminder_email", processor.ProcessTaskSendReminderEmail)

	return processor.server.Start(mux)
}

// Shutdown gracefully waits for running tasks to finish.
func (processor *RedisTaskProcessor) Shutdown() {
	processor.server.Shutdown()
}
