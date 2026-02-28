// Package frontend provides the built-in web UI for AgentSpec agents.
package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Event string      `json:"event"`
	Data  interface{} `json:"data"`
}

// SSEWriter wraps an http.ResponseWriter for SSE streaming.
type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

// NewSSEWriter creates a new SSE writer, setting appropriate headers.
func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	return &SSEWriter{w: w, flusher: flusher}, nil
}

// WriteEvent sends a named SSE event.
func (s *SSEWriter) WriteEvent(event string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(s.w, "event: %s\n", event)
	_, _ = fmt.Fprintf(s.w, "data: %s\n\n", jsonData)
	s.flusher.Flush()
	return nil
}

// WriteToken sends a streaming token event.
func (s *SSEWriter) WriteToken(token string) error {
	return s.WriteEvent("token", map[string]string{"content": token})
}

// WriteThought sends a thought/reasoning event.
func (s *SSEWriter) WriteThought(thought string) error {
	return s.WriteEvent("thought", map[string]string{"content": thought})
}

// WriteToolCall sends a tool invocation event.
func (s *SSEWriter) WriteToolCall(toolName string, args map[string]interface{}) error {
	return s.WriteEvent("tool_call", map[string]interface{}{
		"tool": toolName,
		"args": args,
	})
}

// WriteToolResult sends a tool result event.
func (s *SSEWriter) WriteToolResult(toolName, result string) error {
	return s.WriteEvent("tool_result", map[string]interface{}{
		"tool":   toolName,
		"result": result,
	})
}

// WriteValidation sends a validation event.
func (s *SSEWriter) WriteValidation(rule, status, message string) error {
	return s.WriteEvent("validation", map[string]string{
		"rule":    rule,
		"status":  status,
		"message": message,
	})
}

// WriteDone sends the done event, signaling the stream is complete.
func (s *SSEWriter) WriteDone(finalMessage string) error {
	return s.WriteEvent("done", map[string]string{"message": finalMessage})
}

// WriteError sends an error event.
func (s *SSEWriter) WriteError(message string) error {
	return s.WriteEvent("error", map[string]string{"message": message})
}
