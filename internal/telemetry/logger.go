package telemetry

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

// NewLogger creates a structured JSON logger with default fields.
func NewLogger(w io.Writer, level slog.Level) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level: level,
	})
	return slog.New(handler)
}

// WithCorrelationID adds a correlation ID to the context.
// If id is empty, a new UUID is generated.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	if id == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		id = hex.EncodeToString(b)
	}
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationID retrieves the correlation ID from context.
func CorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}

// RequestLogger returns a logger with request-scoped fields.
func RequestLogger(logger *slog.Logger, ctx context.Context, agent string) *slog.Logger {
	attrs := []any{
		slog.String("agent", agent),
	}
	if id := CorrelationID(ctx); id != "" {
		attrs = append(attrs, slog.String("correlation_id", id))
	}
	return logger.With(attrs...)
}
