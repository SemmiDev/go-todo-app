// cmd/worker is a placeholder binary intended to process background tasks
// using hibiken/asynq (Redis-backed queue).
//
// Unlike cmd/scheduler which triggers time-based jobs, cmd/worker will handle
// ad-hoc asynchronous tasks triggered by the API (e.g. processing large files,
// generating reports, or pushing webhooks).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/semmidev/todo-app/internal/adapter/driven/postgres"
	smtpadapter "github.com/semmidev/todo-app/internal/adapter/driven/smtp"
	"github.com/semmidev/todo-app/internal/adapter/driving/worker"
	"github.com/semmidev/todo-app/internal/common/logging"
	"github.com/semmidev/todo-app/internal/config"
)

func main() {
	cfg := config.Load()

	logger := logging.NewLogger(logging.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.Environment,
		ServiceName: "worker",
	})
	slog.SetDefault(logger)

	logger.Info("worker binary starting", slog.String("redis_url", cfg.RedisURL))

	ctx := context.Background()

	// ── Infrastructure ────────────────────────────────────────────────────────
	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer db.Close()

	todoRepo := postgres.NewTodoRepo(db)
	userRepo := postgres.NewUserRepo(db)

	emailSender, err := smtpadapter.NewSender(smtpadapter.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUsername,
		Password: cfg.SMTPPassword,
		From:     cfg.SMTPFrom,
	})
	if err != nil {
		logger.Error("init smtp sender", slog.Any("error", err))
		os.Exit(1)
	}

	// ── Asynq Worker Setup ────────────────────────────────────────────────────
	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisURL}

	taskProcessor := worker.NewRedisTaskProcessor(
		redisOpt,
		todoRepo,
		userRepo,
		emailSender,
		logger,
		cfg.AppURL,
	)

	// Start processing in a non-blocking way
	errChan := make(chan error, 1)
	go func() {
		if err := taskProcessor.Start(); err != nil {
			errChan <- err
		}
	}()

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	logger.Info("worker running — waiting for signal")

	select {
	case err := <-errChan:
		logger.Error("worker crashed", slog.Any("error", err))
		os.Exit(1)
	case <-quit:
		logger.Info("worker stopping cleanly")
	}

	taskProcessor.Shutdown()
	logger.Info("worker stopped")
}
