package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

// AnthropicClient implements Client using the Anthropic Messages API.
type AnthropicClient struct {
	client anthropic.Client
}

// NewAnthropicClient creates a client that reads ANTHROPIC_API_KEY from the environment.
func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{
		client: anthropic.NewClient(),
	}
}

// NewAnthropicClientWithKey creates a client with an explicit API key.
func NewAnthropicClientWithKey(apiKey string) *AnthropicClient {
	return &AnthropicClient{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
	}
}

// Chat sends a non-streaming chat request.
func (c *AnthropicClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	params := c.buildParams(req)

	msg, err := c.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic chat: %w", err)
	}

	return c.parseResponse(msg), nil
}

// ChatStream sends a streaming chat request and returns events via channel.
func (c *AnthropicClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	params := c.buildParams(req)

	stream := c.client.Messages.NewStreaming(ctx, params)

	ch := make(chan StreamEvent, 64)
	go func() {
		defer close(ch)
		var accMsg anthropic.Message

		for stream.Next() {
			event := stream.Current()
			_ = accMsg.Accumulate(event)

			switch event.Type {
			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					ch <- StreamEvent{Type: "text", Text: event.Delta.Text}
				}
			case "content_block_start":
				if event.ContentBlock.Type == "tool_use" {
					ch <- StreamEvent{
						Type: "tool_call_start",
						ToolCall: &ToolCall{
							ID:   event.ContentBlock.ID,
							Name: event.ContentBlock.Name,
						},
					}
				}
			}
		}

		if err := stream.Err(); err != nil {
			ch <- StreamEvent{Type: "error", Error: err}
			return
		}

		resp := c.parseResponse(&accMsg)
		ch <- StreamEvent{Type: "done", Response: resp}
	}()

	return ch, nil
}

func (c *AnthropicClient) buildParams(req ChatRequest) anthropic.MessageNewParams {
	messages := make([]anthropic.MessageParam, 0, len(req.Messages))
	for _, m := range req.Messages {
		switch m.Role {
		case RoleUser:
			if m.ToolResult != nil {
				messages = append(messages, anthropic.NewUserMessage(
					anthropic.NewToolResultBlock(m.ToolResult.ToolUseID, m.ToolResult.Content, m.ToolResult.IsError),
				))
			} else {
				messages = append(messages, anthropic.NewUserMessage(
					anthropic.NewTextBlock(m.Content),
				))
			}
		case RoleAssistant:
			if len(m.ToolCalls) > 0 {
				blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.ToolCalls)+1)
				if m.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(m.Content))
				}
				for _, tc := range m.ToolCalls {
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, tc.Input, tc.Name))
				}
				messages = append(messages, anthropic.NewAssistantMessage(blocks...))
			} else {
				messages = append(messages, anthropic.NewAssistantMessage(
					anthropic.NewTextBlock(m.Content),
				))
			}
		}
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		Messages:  messages,
		MaxTokens: int64(req.MaxTokens),
	}

	if req.System != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: req.System},
		}
	}

	if req.Temperature != nil {
		params.Temperature = param.NewOpt(*req.Temperature)
	}

	if len(req.Tools) > 0 {
		tools := make([]anthropic.ToolUnionParam, len(req.Tools))
		for i, t := range req.Tools {
			schemaBytes, err := json.Marshal(t.InputSchema)
			if err != nil {
				slog.Warn("anthropic: failed to marshal tool input schema", "tool", t.Name, "error", err)
				continue
			}
			tools[i] = anthropic.ToolUnionParam{
				OfTool: &anthropic.ToolParam{
					Name:        t.Name,
					Description: param.NewOpt(t.Description),
					InputSchema: anthropic.ToolInputSchemaParam{
						Properties: json.RawMessage(schemaBytes),
					},
				},
			}
		}
		params.Tools = tools
	}

	return params
}

func (c *AnthropicClient) parseResponse(msg *anthropic.Message) *ChatResponse {
	resp := &ChatResponse{
		StopReason: mapStopReason(msg.StopReason),
		Usage: TokenUsage{
			InputTokens:  int(msg.Usage.InputTokens),
			OutputTokens: int(msg.Usage.OutputTokens),
			CacheRead:    int(msg.Usage.CacheReadInputTokens),
			CacheWrite:   int(msg.Usage.CacheCreationInputTokens),
		},
	}

	for _, block := range msg.Content {
		switch block.Type {
		case "text":
			resp.Content += block.Text
		case "tool_use":
			input := make(map[string]interface{})
			if err := json.Unmarshal(block.Input, &input); err != nil {
				slog.Warn("anthropic: failed to unmarshal tool input", "tool", block.Name, "id", block.ID, "error", err)
				resp.ToolCalls = append(resp.ToolCalls, ToolCall{
					ID:    block.ID,
					Name:  block.Name,
					Input: map[string]interface{}{"_error": fmt.Sprintf("failed to parse tool input: %v", err)},
				})
			} else {
				resp.ToolCalls = append(resp.ToolCalls, ToolCall{
					ID:    block.ID,
					Name:  block.Name,
					Input: input,
				})
			}
		}
	}

	return resp
}

func mapStopReason(reason anthropic.StopReason) StopReason {
	switch reason {
	case anthropic.StopReasonEndTurn:
		return StopEndTurn
	case anthropic.StopReasonMaxTokens:
		return StopMaxTokens
	case anthropic.StopReasonToolUse:
		return StopToolUse
	case anthropic.StopReasonStopSequence:
		return StopStopSequence
	default:
		return StopReason(string(reason))
	}
}
