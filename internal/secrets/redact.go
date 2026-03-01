package secrets

import (
	"context"
	"log/slog"
	"strings"
	"sync"
)

// RedactFilter wraps a slog handler to scrub resolved secret values from log output.
type RedactFilter struct {
	inner   slog.Handler
	mu      *sync.RWMutex
	secrets map[string]bool
}

// NewRedactFilter creates a log handler that redacts known secret values.
func NewRedactFilter(inner slog.Handler) *RedactFilter {
	return &RedactFilter{
		inner:   inner,
		mu:      &sync.RWMutex{},
		secrets: make(map[string]bool),
	}
}

// AddSecret registers a value to be redacted from log output.
func (f *RedactFilter) AddSecret(value string) {
	if value == "" {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.secrets[value] = true
}

// Enabled delegates to the inner handler.
func (f *RedactFilter) Enabled(ctx context.Context, level slog.Level) bool {
	return f.inner.Enabled(ctx, level)
}

// Handle redacts secret values from log record attributes.
func (f *RedactFilter) Handle(ctx context.Context, record slog.Record) error {
	f.mu.RLock()
	secrets := make([]string, 0, len(f.secrets))
	for s := range f.secrets {
		secrets = append(secrets, s)
	}
	f.mu.RUnlock()

	if len(secrets) == 0 {
		return f.inner.Handle(ctx, record)
	}

	// Redact the message
	msg := record.Message
	for _, s := range secrets {
		msg = strings.ReplaceAll(msg, s, "***REDACTED***")
	}

	// Create new record with redacted message
	redacted := slog.NewRecord(record.Time, record.Level, msg, record.PC)

	// Redact attribute values
	record.Attrs(func(a slog.Attr) bool {
		redacted.AddAttrs(f.redactAttr(a, secrets))
		return true
	})

	return f.inner.Handle(ctx, redacted)
}

// WithAttrs delegates to the inner handler.
// Shares the parent's mutex and secrets map so AddSecret is race-free.
func (f *RedactFilter) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &RedactFilter{
		inner:   f.inner.WithAttrs(attrs),
		mu:      f.mu,
		secrets: f.secrets,
	}
}

// WithGroup delegates to the inner handler.
// Shares the parent's mutex and secrets map so AddSecret is race-free.
func (f *RedactFilter) WithGroup(name string) slog.Handler {
	return &RedactFilter{
		inner:   f.inner.WithGroup(name),
		mu:      f.mu,
		secrets: f.secrets,
	}
}

func (f *RedactFilter) redactAttr(a slog.Attr, secrets []string) slog.Attr {
	if a.Value.Kind() == slog.KindString {
		val := a.Value.String()
		for _, s := range secrets {
			val = strings.ReplaceAll(val, s, "***REDACTED***")
		}
		return slog.String(a.Key, val)
	}
	return a
}

// RedactString replaces any known secret values in a string with a placeholder.
func (f *RedactFilter) RedactString(s string) string {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for secret := range f.secrets {
		s = strings.ReplaceAll(s, secret, "***REDACTED***")
	}
	return s
}
