package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
)

// HTTPConfig configures an HTTP tool executor.
type HTTPConfig struct {
	Method       string            `json:"method"`
	URL          string            `json:"url"`
	Headers      map[string]string `json:"headers,omitempty"`
	BodyTemplate string            `json:"body_template,omitempty"`
}

// HTTPExecutor executes tools via HTTP requests.
type HTTPExecutor struct {
	config HTTPConfig
	client *http.Client
}

// NewHTTPExecutor creates an HTTP tool executor.
func NewHTTPExecutor(config HTTPConfig) *HTTPExecutor {
	return &HTTPExecutor{
		config: config,
		client: http.DefaultClient,
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
		tmpl, err := template.New("body").Parse(e.config.BodyTemplate)
		if err != nil {
			return "", fmt.Errorf("http tool: invalid body template: %w", err)
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, input); err != nil {
			return "", fmt.Errorf("http tool: template execution failed: %w", err)
		}
		body = &buf
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("http tool: read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("http tool: status %d: %s", resp.StatusCode, string(respBody))
	}

	return string(respBody), nil
}
