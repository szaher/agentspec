package telemetry

import (
	"context"
	"time"
)

// Span represents a single trace span for an operation.
type Span struct {
	TraceID   string            `json:"trace_id"`
	SpanID    string            `json:"span_id"`
	ParentID  string            `json:"parent_id,omitempty"`
	Operation string            `json:"operation"`
	StartTime time.Time         `json:"start_time"`
	EndTime   time.Time         `json:"end_time,omitempty"`
	Duration  time.Duration     `json:"duration_ms,omitempty"`
	Status    string            `json:"status"`
	Tags      map[string]string `json:"tags,omitempty"`
}

// Tracer creates and manages trace spans.
type Tracer struct {
	// Exporter receives completed spans. If nil, spans are discarded.
	Exporter SpanExporter
}

// SpanExporter receives completed spans for export to a tracing backend.
type SpanExporter interface {
	ExportSpan(span Span)
}

// SpanExporterFunc is a function adapter for SpanExporter.
type SpanExporterFunc func(span Span)

// ExportSpan calls the function.
func (f SpanExporterFunc) ExportSpan(span Span) { f(span) }

// NewTracer creates a new tracer with an optional exporter.
func NewTracer(exporter SpanExporter) *Tracer {
	return &Tracer{Exporter: exporter}
}

type traceContextKey struct{}

// StartSpan creates a new span and adds it to the context.
func (t *Tracer) StartSpan(ctx context.Context, operation string, tags map[string]string) (context.Context, *Span) {
	span := &Span{
		TraceID:   generateID(),
		SpanID:    generateID(),
		Operation: operation,
		StartTime: time.Now(),
		Status:    "ok",
		Tags:      tags,
	}

	// Inherit trace ID and set parent from context
	if parent, ok := ctx.Value(traceContextKey{}).(*Span); ok {
		span.TraceID = parent.TraceID
		span.ParentID = parent.SpanID
	}

	return context.WithValue(ctx, traceContextKey{}, span), span
}

// EndSpan completes a span and exports it.
func (t *Tracer) EndSpan(span *Span, status string) {
	span.EndTime = time.Now()
	span.Duration = span.EndTime.Sub(span.StartTime)
	if status != "" {
		span.Status = status
	}
	if t.Exporter != nil {
		t.Exporter.ExportSpan(*span)
	}
}

// InvocationTags returns standard tags for an agent invocation span.
func InvocationTags(agent, model, strategy string) map[string]string {
	return map[string]string{
		"agent":    agent,
		"model":    model,
		"strategy": strategy,
	}
}

// LLMCallTags returns standard tags for an LLM call span.
func LLMCallTags(model string, inputTokens, outputTokens int) map[string]string {
	return map[string]string{
		"model":         model,
		"input_tokens":  intToStr(inputTokens),
		"output_tokens": intToStr(outputTokens),
	}
}

// ToolCallTags returns standard tags for a tool call span.
func ToolCallTags(tool, status string) map[string]string {
	return map[string]string{
		"tool":   tool,
		"status": status,
	}
}

func generateID() string {
	// Use correlation ID if available, otherwise generate a simple ID
	// In production, use a proper trace ID generator (e.g., W3C trace context)
	id := make([]byte, 8)
	for i := range id {
		id[i] = "0123456789abcdef"[time.Now().UnixNano()%16]
		time.Sleep(time.Nanosecond)
	}
	return string(id)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
