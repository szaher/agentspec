package loop

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
)

// MapReduceStrategy splits input into chunks, fans out to parallel agent calls,
// and merges results into a single output.
type MapReduceStrategy struct {
	// ChunkSize is the approximate size of each chunk in characters.
	// If 0, the input is split by newlines.
	ChunkSize int
}

// Name returns the strategy identifier.
func (s *MapReduceStrategy) Name() string { return "map-reduce" }

// Execute splits input, maps each chunk to a ReAct invocation, and reduces results.
func (s *MapReduceStrategy) Execute(ctx context.Context, inv Invocation, llmClient llm.Client, tools ToolExecutor, onEvent StreamCallback) (*Response, error) {
	start := time.Now()
	tracker := llm.NewTokenTracker(inv.TokenBudget)

	maxTokens := inv.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	// Split input into chunks
	chunks := s.splitInput(inv.Input)
	if len(chunks) <= 1 {
		// Single chunk, just run ReAct
		react := &ReActStrategy{}
		return react.Execute(ctx, inv, llmClient, tools, onEvent)
	}

	if onEvent != nil {
		onEvent(llm.StreamEvent{Type: "text", Text: fmt.Sprintf("[Map-Reduce: Processing %d chunks]\n", len(chunks))})
	}

	// Map phase: process each chunk concurrently
	type chunkResult struct {
		index  int
		output string
		tokens llm.TokenUsage
		err    error
	}

	results := make([]chunkResult, len(chunks))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var allToolRecords []ToolCallRecord

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, input string) {
			defer wg.Done()

			chunkInv := inv
			chunkInv.Input = input
			chunkInv.Stream = false

			react := &ReActStrategy{}
			resp, err := react.Execute(ctx, chunkInv, llmClient, tools, nil)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				results[idx] = chunkResult{index: idx, err: err}
				cancel()
				return
			}
			results[idx] = chunkResult{
				index:  idx,
				output: resp.Output,
				tokens: resp.Tokens,
			}
			allToolRecords = append(allToolRecords, resp.ToolCalls...)
		}(i, chunk)
	}
	wg.Wait()

	// Check for errors
	var mapOutputs []string
	for _, r := range results {
		if r.err != nil {
			return nil, fmt.Errorf("map-reduce chunk %d: %w", r.index, r.err)
		}
		tracker.Add(r.tokens)
		mapOutputs = append(mapOutputs, r.output)
	}

	// Reduce phase: merge results
	reducePrompt := fmt.Sprintf("%s\n\nI processed the input in %d parts. Here are the results:\n\n",
		inv.System, len(mapOutputs))
	for i, output := range mapOutputs {
		reducePrompt += fmt.Sprintf("--- Part %d ---\n%s\n\n", i+1, output)
	}
	reducePrompt += "Synthesize these results into a single coherent response."

	reduceResp, err := llmClient.Chat(ctx, llm.ChatRequest{
		Model:     inv.Model,
		Messages:  []llm.Message{{Role: llm.RoleUser, Content: reducePrompt}},
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("map-reduce reduce: %w", err)
	}
	tracker.Add(reduceResp.Usage)

	return &Response{
		Output:    reduceResp.Content,
		ToolCalls: allToolRecords,
		Tokens:    tracker.Usage(),
		Turns:     len(chunks) + 2, // map chunks + plan + reduce
		Duration:  time.Since(start),
	}, nil
}

func (s *MapReduceStrategy) splitInput(input string) []string {
	chunkSize := s.ChunkSize
	if chunkSize <= 0 {
		// Split by double newlines (paragraphs)
		parts := strings.Split(input, "\n\n")
		var chunks []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				chunks = append(chunks, p)
			}
		}
		if len(chunks) == 0 {
			return []string{input}
		}
		return chunks
	}

	// Split by character count
	var chunks []string
	for i := 0; i < len(input); i += chunkSize {
		end := i + chunkSize
		if end > len(input) {
			end = len(input)
		}
		chunks = append(chunks, input[i:end])
	}
	return chunks
}
