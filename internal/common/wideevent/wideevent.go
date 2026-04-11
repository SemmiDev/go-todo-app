// Package wideevent implements the "canonical log line" / "wide event" pattern
// described at https://loggingsucks.com
//
// Instead of many narrow log lines scattered across a request, one rich event
// is built up progressively and emitted as a single log line at the end of the
// request. Every layer (interceptors, handlers, services) can contribute fields
// without emitting additional log lines.
//
// Usage:
//
//	// In an interceptor — initialise at the top:
//	ctx = wideevent.New(ctx)
//
//	// Anywhere inside the request lifecycle — enrich:
//	wideevent.Add(ctx, slog.String("user_id", u.ID().String()))
//	wideevent.Add(ctx, slog.Int("tag_count", len(tags)))
//
//	// Back in the interceptor — flush once at the very end:
//	wideevent.Flush(ctx, logger, "grpc request", slog.LevelInfo)
package wideevent

import (
	"context"
	"log/slog"
	"sync"
)

type eventKey struct{}

// Event is a thread-safe, context-carried accumulator of slog attributes.
// It is allocated once per request and enriched by any layer that has access
// to the context.
type Event struct {
	mu    sync.Mutex
	attrs []slog.Attr
}

// New injects a fresh, empty Event into the context and returns the enriched
// context. Call this at the very beginning of a request (e.g. in a gRPC
// interceptor).
func New(ctx context.Context) context.Context {
	return context.WithValue(ctx, eventKey{}, &Event{})
}

// Add appends one or more slog.Attr values to the wide event stored in ctx.
// It is a no-op if no event has been initialised in ctx (safe to call
// everywhere without nil-checking).
func Add(ctx context.Context, attrs ...slog.Attr) {
	e, ok := ctx.Value(eventKey{}).(*Event)
	if !ok || e == nil {
		return
	}
	e.mu.Lock()
	e.attrs = append(e.attrs, attrs...)
	e.mu.Unlock()
}

// Flush emits the accumulated wide event as a single structured log record via
// the provided logger. Call this once, at the very end of the request (e.g.
// the deferred section of a gRPC interceptor).
//
// All attributes that were added via Add are attached to this single record —
// zero additional lines are emitted.
func Flush(ctx context.Context, logger *slog.Logger, msg string, level slog.Level) {
	e, ok := ctx.Value(eventKey{}).(*Event)
	if !ok || e == nil {
		return
	}
	e.mu.Lock()
	attrs := e.attrs
	e.mu.Unlock()

	// Convert []slog.Attr → []any so we can use the variadic slog.Logger.Log.
	args := make([]any, 0, len(attrs))
	for _, a := range attrs {
		args = append(args, a)
	}
	logger.Log(ctx, level, msg, args...)
}
