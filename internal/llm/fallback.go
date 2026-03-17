package llm

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// FallbackClient wraps multiple LLM clients and tries them in order.
// If the primary client fails, it falls back to the next client in the chain.
type FallbackClient struct {
	clients []Client
	models  []string
	logger  *slog.Logger
	// OnFallback is called when a fallback occurs (for metrics recording)
	OnFallback func(fromModel, toModel string)
}

// NewFallbackClient creates a client that tries each model in order.
func NewFallbackClient(clients []Client, models []string, logger *slog.Logger) *FallbackClient {
	return &FallbackClient{
		clients: clients,
		models:  models,
		logger:  logger,
	}
}

// Chat sends a request and returns the complete response, trying each client
// in order until one succeeds. If all clients fail, an error listing all
// failures is returned.
func (f *FallbackClient) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var errs []string

	for i, client := range f.clients {
		model := f.modelName(i)
		resp, err := client.Chat(ctx, req)
		if err == nil {
			return resp, nil
		}

		f.logger.WarnContext(ctx, "LLM chat failed, attempting fallback",
			slog.String("model", model),
			slog.String("error", err.Error()),
		)
		errs = append(errs, fmt.Sprintf("%s: %v", model, err))

		if i+1 < len(f.clients) && f.OnFallback != nil {
			f.OnFallback(model, f.modelName(i+1))
		}
	}

	return nil, fmt.Errorf("all LLM clients failed: %s", strings.Join(errs, "; "))
}

// ChatStream sends a request and returns a channel of streaming events, trying
// each client in order until one succeeds. If all clients fail, an error
// listing all failures is returned.
func (f *FallbackClient) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	var errs []string

	for i, client := range f.clients {
		model := f.modelName(i)
		ch, err := client.ChatStream(ctx, req)
		if err == nil {
			return ch, nil
		}

		f.logger.WarnContext(ctx, "LLM chat stream failed, attempting fallback",
			slog.String("model", model),
			slog.String("error", err.Error()),
		)
		errs = append(errs, fmt.Sprintf("%s: %v", model, err))

		if i+1 < len(f.clients) && f.OnFallback != nil {
			f.OnFallback(model, f.modelName(i+1))
		}
	}

	return nil, fmt.Errorf("all LLM clients failed: %s", strings.Join(errs, "; "))
}

// modelName returns the model name for the client at the given index.
func (f *FallbackClient) modelName(i int) string {
	if i < len(f.models) {
		return f.models[i]
	}
	return fmt.Sprintf("client-%d", i)
}
