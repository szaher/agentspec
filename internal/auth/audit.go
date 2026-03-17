package auth

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

// AuditEntry records a single agent invocation for audit purposes.
type AuditEntry struct {
	Timestamp     string `json:"timestamp"`
	User          string `json:"user"`
	Agent         string `json:"agent"`
	SessionID     string `json:"session_id,omitempty"`
	Action        string `json:"action"` // "invoke", "stream"
	InputTokens   int    `json:"input_tokens"`
	OutputTokens  int    `json:"output_tokens"`
	DurationMs    int64  `json:"duration_ms"`
	Status        string `json:"status"` // "success", "error", "denied"
	CorrelationID string `json:"correlation_id,omitempty"`
	Error         string `json:"error,omitempty"`
}

// AuditLogger writes JSON-line audit log entries to a file.
type AuditLogger struct {
	mu     sync.Mutex
	file   *os.File
	logger *slog.Logger
}

// NewAuditLogger creates an audit logger writing to the given file path.
func NewAuditLogger(path string, logger *slog.Logger) (*AuditLogger, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, err
	}
	return &AuditLogger{file: f, logger: logger}, nil
}

// Log writes an audit entry as a JSON line.
func (a *AuditLogger) Log(entry AuditEntry) {
	if entry.Timestamp == "" {
		entry.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	data, err := json.Marshal(entry)
	if err != nil {
		a.logger.Error("failed to marshal audit entry", "error", err)
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, _ = a.file.Write(append(data, '\n'))
}

// Close closes the audit log file.
func (a *AuditLogger) Close() error {
	if a.file != nil {
		return a.file.Close()
	}
	return nil
}
