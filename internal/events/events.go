// Package events defines structured event types for the
// Agentz toolchain lifecycle operations.
package events

import (
	"encoding/json"
	"time"
)

// Type represents the kind of event.
type Type string

const (
	PlanStarted    Type = "plan.started"
	ApplyStarted   Type = "apply.started"
	ApplyResource  Type = "apply.resource"
	ApplyCompleted Type = "apply.completed"
	RunStarted     Type = "run.started"
	RunProgress    Type = "run.progress"
	RunCompleted   Type = "run.completed"
	RunFailed      Type = "run.failed"
)

// Event is a structured event emitted during toolchain operations.
type Event struct {
	Type          Type                   `json:"type"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id"`
	Data          map[string]interface{} `json:"data,omitempty"`
}

// New creates a new event with the given type and correlation ID.
func New(eventType Type, correlationID string) *Event {
	return &Event{
		Type:          eventType,
		Timestamp:     time.Now(),
		CorrelationID: correlationID,
	}
}

// WithData adds data fields to the event and returns it for chaining.
func (e *Event) WithData(key string, value interface{}) *Event {
	if e.Data == nil {
		e.Data = make(map[string]interface{})
	}
	e.Data[key] = value
	return e
}

// JSON returns the event serialized as JSON.
func (e *Event) JSON() ([]byte, error) {
	return json.Marshal(e)
}

// Emitter is the interface for event consumers.
type Emitter interface {
	Emit(event *Event)
}

// NoopEmitter discards all events.
type NoopEmitter struct{}

// Emit implements Emitter by discarding the event.
func (NoopEmitter) Emit(*Event) {}

// CollectorEmitter collects events in memory for testing.
type CollectorEmitter struct {
	Events []*Event
}

// Emit appends the event to the collector.
func (c *CollectorEmitter) Emit(event *Event) {
	c.Events = append(c.Events, event)
}
