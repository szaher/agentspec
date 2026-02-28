package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OpenAIClient implements Client using the OpenAI-compatible chat completions API.
// Works with Ollama, OpenAI, vLLM, LiteLLM, and any OpenAI-compatible endpoint.
type OpenAIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// OpenAIOption configures the OpenAI client.
type OpenAIOption func(*OpenAIClient)

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(c *http.Client) OpenAIOption {
	return func(o *OpenAIClient) { o.httpClient = c }
}

// NewOpenAIClient creates a client for the OpenAI API.
func NewOpenAIClient(apiKey string, opts ...OpenAIOption) *OpenAIClient {
	c := &OpenAIClient{
		baseURL:    "https://api.openai.com/v1",
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewOllamaClient creates a client for a local Ollama instance.
func NewOllamaClient(host string, opts ...OpenAIOption) *OpenAIClient {
	if host == "" {
		host = "http://localhost:11434"
	}
	c := &OpenAIClient{
		baseURL:    strings.TrimRight(host, "/") + "/v1",
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// NewOpenAICompatibleClient creates a client for any OpenAI-compatible endpoint.
func NewOpenAICompatibleClient(baseURL, apiKey string, opts ...OpenAIOption) *OpenAIClient {
	c := &OpenAIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// --- OpenAI API request/response types ---

type oaiRequest struct {
	Model       string        `json:"model"`
	Messages    []oaiMessage  `json:"messages"`
	Tools       []oaiTool     `json:"tools,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature *float64      `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type oaiMessage struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCalls  []oaiToolCall  `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type oaiTool struct {
	Type     string      `json:"type"`
	Function oaiFunction `json:"function"`
}

type oaiFunction struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

type oaiToolCall struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Function oaiToolCallFunc `json:"function"`
}

type oaiToolCallFunc struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type oaiResponse struct {
	Choices []oaiChoice `json:"choices"`
	Usage   oaiUsage    `json:"usage"`
	Error   *oaiError   `json:"error,omitempty"`
}

type oaiChoice struct {
	Message      oaiMessage `json:"message"`
	Delta        oaiMessage `json:"delta"`
	FinishReason string     `json:"finish_reason"`
}

type oaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type oaiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

// Chat sends a non-streaming chat request.
func (c *OpenAIClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	oaiReq := c.buildRequest(req, false)

	body, err := c.doRequest(ctx, oaiReq)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	var oaiResp oaiResponse
	if err := json.NewDecoder(body).Decode(&oaiResp); err != nil {
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}

	if oaiResp.Error != nil {
		return nil, fmt.Errorf("openai: %s: %s", oaiResp.Error.Type, oaiResp.Error.Message)
	}

	return c.parseResponse(&oaiResp), nil
}

// ChatStream sends a streaming chat request and returns events via channel.
func (c *OpenAIClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	oaiReq := c.buildRequest(req, true)

	body, err := c.doRequest(ctx, oaiReq)
	if err != nil {
		return nil, err
	}

	ch := make(chan StreamEvent, 64)
	go func() {
		defer close(ch)
		defer body.Close()

		var fullContent strings.Builder
		var toolCalls []ToolCall
		var usage oaiUsage
		var finishReason string

		scanner := bufio.NewScanner(body)
		for scanner.Scan() {
			line := scanner.Text()

			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk oaiResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if chunk.Usage.TotalTokens > 0 {
				usage = chunk.Usage
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			choice := chunk.Choices[0]
			if choice.FinishReason != "" {
				finishReason = choice.FinishReason
			}

			delta := choice.Delta

			// Text content
			if delta.Content != "" {
				fullContent.WriteString(delta.Content)
				ch <- StreamEvent{Type: "text", Text: delta.Content}
			}

			// Tool calls
			for _, tc := range delta.ToolCalls {
				if tc.Function.Name != "" {
					ch <- StreamEvent{
						Type: "tool_call_start",
						ToolCall: &ToolCall{
							ID:   tc.ID,
							Name: tc.Function.Name,
						},
					}
				}
				if tc.Function.Arguments != "" {
					// Accumulate arguments for tool calls
					found := false
					for i := range toolCalls {
						if toolCalls[i].ID == tc.ID {
							// Append partial args (streaming sends fragments)
							found = true
							break
						}
					}
					if !found && tc.ID != "" {
						input := make(map[string]interface{})
						_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
						toolCalls = append(toolCalls, ToolCall{
							ID:    tc.ID,
							Name:  tc.Function.Name,
							Input: input,
						})
					}
				}
			}
		}

		resp := &ChatResponse{
			Content:    fullContent.String(),
			ToolCalls:  toolCalls,
			StopReason: mapOAIStopReason(finishReason),
			Usage: TokenUsage{
				InputTokens:  usage.PromptTokens,
				OutputTokens: usage.CompletionTokens,
			},
		}
		ch <- StreamEvent{Type: "done", Response: resp}
	}()

	return ch, nil
}

func (c *OpenAIClient) buildRequest(req ChatRequest, stream bool) oaiRequest {
	messages := make([]oaiMessage, 0, len(req.Messages)+1)

	// System message
	if req.System != "" {
		messages = append(messages, oaiMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	// Conversation messages
	for _, m := range req.Messages {
		msg := oaiMessage{}
		switch m.Role {
		case RoleUser:
			if m.ToolResult != nil {
				msg.Role = "tool"
				msg.Content = m.ToolResult.Content
				msg.ToolCallID = m.ToolResult.ToolUseID
			} else {
				msg.Role = "user"
				msg.Content = m.Content
			}
		case RoleAssistant:
			msg.Role = "assistant"
			msg.Content = m.Content
			for _, tc := range m.ToolCalls {
				args, _ := json.Marshal(tc.Input)
				msg.ToolCalls = append(msg.ToolCalls, oaiToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: oaiToolCallFunc{
						Name:      tc.Name,
						Arguments: string(args),
					},
				})
			}
		}
		messages = append(messages, msg)
	}

	oaiReq := oaiRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   stream,
	}

	if req.MaxTokens > 0 {
		oaiReq.MaxTokens = req.MaxTokens
	}
	if req.Temperature != nil {
		oaiReq.Temperature = req.Temperature
	}

	// Tools
	for _, t := range req.Tools {
		oaiReq.Tools = append(oaiReq.Tools, oaiTool{
			Type: "function",
			Function: oaiFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	return oaiReq
}

func (c *OpenAIClient) doRequest(ctx context.Context, oaiReq oaiRequest) (io.ReadCloser, error) {
	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var oaiErr oaiResponse
		if err := json.NewDecoder(resp.Body).Decode(&oaiErr); err == nil && oaiErr.Error != nil {
			return nil, fmt.Errorf("openai: HTTP %d: %s: %s", resp.StatusCode, oaiErr.Error.Type, oaiErr.Error.Message)
		}
		return nil, fmt.Errorf("openai: HTTP %d", resp.StatusCode)
	}

	return resp.Body, nil
}

func (c *OpenAIClient) parseResponse(resp *oaiResponse) *ChatResponse {
	if len(resp.Choices) == 0 {
		return &ChatResponse{
			StopReason: StopEndTurn,
			Usage: TokenUsage{
				InputTokens:  resp.Usage.PromptTokens,
				OutputTokens: resp.Usage.CompletionTokens,
			},
		}
	}

	choice := resp.Choices[0]
	result := &ChatResponse{
		Content:    choice.Message.Content,
		StopReason: mapOAIStopReason(choice.FinishReason),
		Usage: TokenUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}

	for _, tc := range choice.Message.ToolCalls {
		input := make(map[string]interface{})
		_ = json.Unmarshal([]byte(tc.Function.Arguments), &input)
		result.ToolCalls = append(result.ToolCalls, ToolCall{
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: input,
		})
	}

	return result
}

func mapOAIStopReason(reason string) StopReason {
	switch reason {
	case "stop":
		return StopEndTurn
	case "length":
		return StopMaxTokens
	case "tool_calls":
		return StopToolUse
	default:
		return StopEndTurn
	}
}
