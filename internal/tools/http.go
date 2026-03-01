package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultMaxResponseSize int64 = 10 * 1024 * 1024 // 10MB

// HTTPConfig configures an HTTP tool executor.
type HTTPConfig struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyTemplate string            `json:"body_template,omitempty"`
}

// HTTPExecutor executes tools via HTTP requests.
type HTTPExecutor struct {
	config      HTTPConfig
	client      *http.Client
	maxRespSize int64
}

// NewHTTPExecutor creates an HTTP tool executor with SSRF protection.
func NewHTTPExecutor(config HTTPConfig) *HTTPExecutor {
	return &HTTPExecutor{
		config: config,
		client: &http.Client{
			Transport: NewSafeTransport(),
		},
		maxRespSize: defaultMaxResponseSize,
	}
}

// Execute makes an HTTP request with the given input and returns the response body.
func (e *HTTPExecutor) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	method := strings.ToUpper(e.config.Method)
	if method == "" {
		method = "GET"
	}

	url := e.config.URL

	var body io.Reader
	if e.config.BodyTemplate != "" {
		// Use safe body serialization — no Go template execution
		rendered := SafeBodyString([]byte(e.config.BodyTemplate), "text/plain")
		body = strings.NewReader(rendered)
	} else if method == "POST" || method == "PUT" || method == "PATCH" {
		data, err := json.Marshal(input)
		if err != nil {
			return "", fmt.Errorf("http tool: marshal input: %w", err)
		}
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return "", fmt.Errorf("http tool: create request: %w", err)
	}

	for k, v := range e.config.Headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("Content-Type") == "" && body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http tool: request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, truncated, err := ReadBody(resp.Body, e.maxRespSize)
	if err != nil {
		return "", fmt.Errorf("http tool: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http tool: status %d: %s", resp.StatusCode, string(respBody))
	}

	result := SafeBodyString(respBody, resp.Header.Get("Content-Type"))
	if truncated {
		result += "\n[response body truncated at 10MB limit]"
	}

	return result, nil
}

// ReadBody reads the response body with a size limit.
// Returns (data, truncated, error).
func ReadBody(body io.Reader, limit int64) ([]byte, bool, error) {
	if limit <= 0 {
		limit = defaultMaxResponseSize
	}
	lr := io.LimitReader(body, limit+1) // read one extra byte to detect truncation
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > limit {
		return data[:limit], true, nil
	}
	return data, false, nil
}

// SafeBodyString converts an HTTP response body to a safe string representation.
// Sanitizes {{ and }} sequences to prevent template injection.
func SafeBodyString(body []byte, contentType string) string {
	s := string(body)

	// Sanitize Go template delimiters to prevent injection
	s = strings.ReplaceAll(s, "{{", "{ {")
	s = strings.ReplaceAll(s, "}}", "} }")

	// For HTML content, escape HTML entities
	if strings.Contains(contentType, "text/html") {
		s = strings.ReplaceAll(s, "<", "&lt;")
		s = strings.ReplaceAll(s, ">", "&gt;")
	}

	return s
}
