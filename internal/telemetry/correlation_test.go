package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// ulidPattern matches a valid ULID: exactly 26 uppercase alphanumeric characters.
var ulidPattern = regexp.MustCompile(`^[0-9A-Z]{26}$`)

func TestCorrelationMiddlewareGeneratesID(t *testing.T) {
	var capturedID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = CorrelationID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := CorrelationMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// The response must contain the X-Correlation-ID header.
	headerID := rec.Header().Get("X-Correlation-ID")
	if headerID == "" {
		t.Fatal("expected X-Correlation-ID header in response, got empty string")
	}

	// The generated ID must be a valid ULID (26 uppercase alphanumeric chars).
	if !ulidPattern.MatchString(headerID) {
		t.Fatalf("expected a valid ULID (26 uppercase alphanumeric chars), got %q", headerID)
	}

	// The downstream handler must receive the same ID via context.
	if capturedID != headerID {
		t.Fatalf("context correlation ID = %q, response header = %q; expected them to match", capturedID, headerID)
	}
}

func TestCorrelationMiddlewarePreservesID(t *testing.T) {
	const customID = "my-custom-id"
	var capturedID string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedID = CorrelationID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := CorrelationMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Correlation-ID", customID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// The response header must preserve the caller-provided ID.
	headerID := rec.Header().Get("X-Correlation-ID")
	if headerID != customID {
		t.Fatalf("expected response header X-Correlation-ID = %q, got %q", customID, headerID)
	}

	// The downstream handler must receive the same caller-provided ID.
	if capturedID != customID {
		t.Fatalf("expected context correlation ID = %q, got %q", customID, capturedID)
	}
}

func TestCorrelationMiddlewareUniqueIDs(t *testing.T) {
	var ids []string

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, CorrelationID(r.Context()))
		w.WriteHeader(http.StatusOK)
	})

	handler := CorrelationMiddleware(inner)

	// Send two requests without X-Correlation-ID headers.
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
	}

	if len(ids) != 2 {
		t.Fatalf("expected 2 captured IDs, got %d", len(ids))
	}
	if ids[0] == ids[1] {
		t.Fatalf("expected unique IDs for separate requests, but both were %q", ids[0])
	}
}

func TestWithCorrelationID(t *testing.T) {
	t.Run("stores and retrieves a value", func(t *testing.T) {
		ctx := WithCorrelationID(context.Background(), "test-id")
		got := CorrelationID(ctx)
		if got != "test-id" {
			t.Fatalf("expected CorrelationID = %q, got %q", "test-id", got)
		}
	})

	t.Run("plain context returns empty string", func(t *testing.T) {
		got := CorrelationID(context.Background())
		if got != "" {
			t.Fatalf("expected empty string from plain context, got %q", got)
		}
	})

	t.Run("empty id generates a value", func(t *testing.T) {
		ctx := WithCorrelationID(context.Background(), "")
		got := CorrelationID(ctx)
		if got == "" {
			t.Fatal("expected a generated correlation ID when empty string is passed, got empty string")
		}
	})
}

func TestRequestLogger(t *testing.T) {
	var buf bytes.Buffer

	base := NewLogger(&buf, slog.LevelInfo)

	ctx := WithCorrelationID(context.Background(), "req-123")
	logger := RequestLogger(base, ctx, "my-agent")

	logger.Info("hello")

	// Parse the JSON log line.
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v\nraw output: %s", err, buf.String())
	}

	// Verify the "correlation_id" field.
	if cid, ok := entry["correlation_id"]; !ok {
		t.Fatalf("expected 'correlation_id' field in log output, got keys: %v", keys(entry))
	} else if cid != "req-123" {
		t.Fatalf("expected correlation_id = %q, got %q", "req-123", cid)
	}

	// Verify the "agent" field.
	if agent, ok := entry["agent"]; !ok {
		t.Fatalf("expected 'agent' field in log output, got keys: %v", keys(entry))
	} else if agent != "my-agent" {
		t.Fatalf("expected agent = %q, got %q", "my-agent", agent)
	}
}

func TestRequestLoggerWithoutCorrelationID(t *testing.T) {
	var buf bytes.Buffer

	base := NewLogger(&buf, slog.LevelInfo)

	// Use a plain context with no correlation ID set.
	logger := RequestLogger(base, context.Background(), "agent-x")

	logger.Info("no correlation")

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log JSON: %v\nraw output: %s", err, buf.String())
	}

	// The "agent" field should still be present.
	if agent, ok := entry["agent"]; !ok {
		t.Fatalf("expected 'agent' field in log output, got keys: %v", keys(entry))
	} else if agent != "agent-x" {
		t.Fatalf("expected agent = %q, got %q", "agent-x", agent)
	}

	// The "correlation_id" field should NOT be present.
	if _, ok := entry["correlation_id"]; ok {
		t.Fatal("expected no 'correlation_id' field when context has no correlation ID")
	}
}

// keys returns the keys of a map for diagnostic output.
func keys(m map[string]any) []string {
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}
	return result
}
