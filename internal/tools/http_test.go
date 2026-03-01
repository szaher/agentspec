package tools

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// newTestHTTPExecutor creates an HTTPExecutor with a regular http.Client
// (no SSRF protection) so we can test against httptest servers that use
// loopback addresses.
func newTestHTTPExecutor(config HTTPConfig) *HTTPExecutor {
	return &HTTPExecutor{
		config:      config,
		client:      &http.Client{},
		maxRespSize: 10 * 1024 * 1024,
	}
}

// ---- ReadBody tests ----

func TestReadBody_WithinLimit(t *testing.T) {
	body := strings.NewReader("hello world")
	data, truncated, err := ReadBody(body, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatal("expected truncated=false, got true")
	}
	if string(data) != "hello world" {
		t.Fatalf("expected %q, got %q", "hello world", string(data))
	}
}

func TestReadBody_ExceedingLimit(t *testing.T) {
	content := strings.Repeat("a", 100)
	body := strings.NewReader(content)
	data, truncated, err := ReadBody(body, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated {
		t.Fatal("expected truncated=true, got false")
	}
	if len(data) != 50 {
		t.Fatalf("expected data length 50, got %d", len(data))
	}
}

func TestReadBody_EmptyBody(t *testing.T) {
	body := strings.NewReader("")
	data, truncated, err := ReadBody(body, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatal("expected truncated=false, got true")
	}
	if len(data) != 0 {
		t.Fatalf("expected empty data, got %d bytes", len(data))
	}
}

func TestReadBody_ZeroLimit(t *testing.T) {
	// When limit is 0, it should use the default (10MB)
	body := strings.NewReader("test data")
	data, truncated, err := ReadBody(body, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatal("expected truncated=false for small body with default limit")
	}
	if string(data) != "test data" {
		t.Fatalf("expected %q, got %q", "test data", string(data))
	}
}

// ---- SafeBodyString tests ----

func TestSafeBodyString_EscapesTemplateDelimiters(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		want        string
	}{
		{
			name:        "double curly braces escaped",
			body:        "Hello {{name}}",
			contentType: "text/plain",
			want:        "Hello { {name} }",
		},
		{
			name:        "multiple template expressions",
			body:        "{{a}} and {{b}}",
			contentType: "text/plain",
			want:        "{ {a} } and { {b} }",
		},
		{
			name:        "no template expressions unchanged",
			body:        "Hello world",
			contentType: "text/plain",
			want:        "Hello world",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := SafeBodyString([]byte(tc.body), tc.contentType)
			if got != tc.want {
				t.Fatalf("SafeBodyString(%q, %q) = %q, want %q", tc.body, tc.contentType, got, tc.want)
			}
		})
	}
}

func TestSafeBodyString_HTMLContentType(t *testing.T) {
	body := "<div>Hello</div>"
	got := SafeBodyString([]byte(body), "text/html; charset=utf-8")

	// Template delimiters are sanitized first, then HTML entities are escaped
	if strings.Contains(got, "<") || strings.Contains(got, ">") {
		t.Fatalf("expected HTML entities to be escaped, got: %q", got)
	}
	if !strings.Contains(got, "&lt;") || !strings.Contains(got, "&gt;") {
		t.Fatalf("expected &lt; and &gt; in output, got: %q", got)
	}
}

func TestSafeBodyString_PlainContentTypeNoHTMLEscape(t *testing.T) {
	body := "<div>Hello</div>"
	got := SafeBodyString([]byte(body), "text/plain")

	// For plain text, < and > should NOT be escaped
	if strings.Contains(got, "&lt;") || strings.Contains(got, "&gt;") {
		t.Fatalf("plain text should not escape HTML entities, got: %q", got)
	}
	if got != "<div>Hello</div>" {
		t.Fatalf("expected unchanged plain text, got: %q", got)
	}
}

// ---- HTTPExecutor.Execute tests ----

func TestHTTPExecutor_GETRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("response body"))
	}))
	defer server.Close()

	executor := newTestHTTPExecutor(HTTPConfig{
		Method: "GET",
		URL:    server.URL,
	})

	result, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "response body" {
		t.Fatalf("expected %q, got %q", "response body", result)
	}
}

func TestHTTPExecutor_POSTRequest(t *testing.T) {
	var receivedBody string
	var receivedContentType string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		receivedContentType = r.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	executor := newTestHTTPExecutor(HTTPConfig{
		Method: "POST",
		URL:    server.URL,
	})

	input := map[string]interface{}{
		"message": "hello",
	}

	result, err := executor.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "ok" {
		t.Fatalf("expected %q, got %q", "ok", result)
	}

	// Verify the request body was received as JSON
	if !strings.Contains(receivedBody, `"message"`) {
		t.Fatalf("expected JSON body with 'message' key, got: %s", receivedBody)
	}
	if receivedContentType != "application/json" {
		t.Fatalf("expected Content-Type application/json, got: %s", receivedContentType)
	}
}

func TestHTTPExecutor_ErrorStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"bad request", http.StatusBadRequest},
		{"not found", http.StatusNotFound},
		{"internal server error", http.StatusInternalServerError},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, _ = w.Write([]byte("error details"))
			}))
			defer server.Close()

			executor := newTestHTTPExecutor(HTTPConfig{
				Method: "GET",
				URL:    server.URL,
			})

			_, err := executor.Execute(context.Background(), nil)
			if err == nil {
				t.Fatalf("expected error for status %d, got nil", tc.statusCode)
			}
			if !strings.Contains(err.Error(), "error details") {
				t.Fatalf("expected error to contain response body, got: %v", err)
			}
		})
	}
}

func TestHTTPExecutor_WithCustomHeaders(t *testing.T) {
	var receivedAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("authenticated"))
	}))
	defer server.Close()

	executor := newTestHTTPExecutor(HTTPConfig{
		Method: "GET",
		URL:    server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer test-token",
		},
	})

	result, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "authenticated" {
		t.Fatalf("expected %q, got %q", "authenticated", result)
	}
	if receivedAuth != "Bearer test-token" {
		t.Fatalf("expected Authorization header %q, got %q", "Bearer test-token", receivedAuth)
	}
}

func TestHTTPExecutor_TruncatedResponse(t *testing.T) {
	largeBody := strings.Repeat("x", 200)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(largeBody))
	}))
	defer server.Close()

	// Create executor with a small max response size to trigger truncation
	executor := &HTTPExecutor{
		config: HTTPConfig{
			Method: "GET",
			URL:    server.URL,
		},
		client:      &http.Client{},
		maxRespSize: 100,
	}

	result, err := executor.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "[response body truncated at 10MB limit]") {
		t.Fatalf("expected truncation message in result, got: %s", result)
	}
}
