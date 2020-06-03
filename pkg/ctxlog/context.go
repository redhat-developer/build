package ctxlog

import (
	"context"

	"github.com/go-logr/logr"
)

type contextLogger struct{}

var (
	loggerKey = &contextLogger{}
)

// NewParentContext returns a new context from the
// parent context.Background one. This new context
// stores our logger implementation
func NewParentContext(log logr.Logger) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, loggerKey, log)
	return ctx
}

// NewContext returns a new child context based on our logger
// key(loggerKey). This function is useful for spawning children
// context with a particular logging name for each controller
func NewContext(ctx context.Context, name string) context.Context {
	log := ExtractLogger(ctx)
	log = log.WithName(name)
	return context.WithValue(ctx, loggerKey, log)
}

// ExtractLogger returns a logger based on the loggerKey
// This function retrieves from an existing context the value,
// which in this case is an instance of our logger
func ExtractLogger(ctx context.Context) logr.Logger {
	log, ok := ctx.Value(loggerKey).(logr.Logger)
	if !ok || log == nil {
		return nil
	}
	return log
}
