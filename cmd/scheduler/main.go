// cmd/scheduler is a separate binary that runs scheduled background jobs.
// It shares all internal/ packages with cmd/server but has its own lifecycle,
// allowing independent deployment, restarts, and exactly-one-replica guarantees.
//
// Current jobs:
//   - Todo reminder emails (robfig/cron/v3, configurable via REMINDER_CRON)
//
// Future jobs can be added by registering additional cron.AddFunc entries.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/asynqtask"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/memcached"
	"github.com/semmidev/go-todo-app/internal/adapter/driven/postgres"
	reminderapp "github.com/semmidev/go-todo-app/internal/application/reminder"
	"github.com/semmidev/go-todo-app/internal/common/logging"
	"github.com/semmidev/go-todo-app/internal/config"
)

func main() {
	cfg := config.Load()

	logger := logging.NewLogger(logging.Config{
		Level:       cfg.LogLevel,
		Environment: cfg.Environment,
		ServiceName: "scheduler",
	})

	slog.SetDefault(logger)
	logger.Info("scheduler starting",
		slog.String("cron", cfg.ReminderCron),
	)

	ctx := context.Background()

	// ── Infrastructure ────────────────────────────────────────────────────────
	db, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect postgres", slog.Any("error", err))
		os.Exit(1)
	}

	todoRepo := postgres.NewTodoRepo(db)

	redisOpt := asynq.RedisClientOpt{Addr: cfg.RedisURL}
	taskDistributor := asynqtask.NewDistributor(redisOpt)

	_ = memcached.NewCacheRepo(cfg.MemcachedURL) // keep memcached alive for shared infra

	reminderSvc := reminderapp.NewService(
		todoRepo,
		taskDistributor,
		logger,
		reminderapp.Config{},
	)

	// ── Cron scheduler ────────────────────────────────────────────────────────
	// cron.WithSeconds() enables 6-field expressions if needed.
	// cron.WithLogger wraps slog so cron internals use our structured logger.
	cronLogger := cron.VerbosePrintfLogger(slog.NewLogLogger(logger.Handler(), slog.LevelDebug))
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLogger(cronLogger),
		cron.WithChain(
			cron.SkipIfStillRunning(cronLogger), // never overlap job runs
			cron.Recover(cronLogger),            // catch panics in job funcs
		),
	)

	if _, err := c.AddFunc(cfg.ReminderCron, func() {
		logger.Info("reminder job: starting")
		if err := reminderSvc.SendDueSoonReminders(ctx); err != nil {
			logger.Error("reminder job: failed", slog.Any("error", err))
		}
	}); err != nil {
		logger.Error("register reminder cron", slog.Any("error", err))
		os.Exit(1)
	}

	c.Start()
	logger.Info("scheduler running — waiting for signal")

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("scheduler stopping — draining running jobs")
	stopCtx := c.Stop() // returns a context that closes when all jobs finish
	<-stopCtx.Done()
	logger.Info("scheduler stopped cleanly")
}
