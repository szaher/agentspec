package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	"github.com/szaher/designs/agentz/internal/memory"
	"github.com/szaher/designs/agentz/internal/runtime"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/tools"
)

func newRunCmd() *cobra.Command {
	var (
		input  string
		agent  string
		stream bool
	)

	cmd := &cobra.Command{
		Use:   "run [file.ias]",
		Short: "Invoke an agent and print the response",
		Long:  "One-shot agent invocation: parse, validate, start runtime, invoke, print response, shutdown.",
		RunE: func(cmd *cobra.Command, args []string) error {
			files, err := resolveFiles(args)
			if err != nil {
				return err
			}

			doc, err := parseAndLower(files)
			if err != nil {
				return err
			}

			config, err := runtime.FromIR(doc)
			if err != nil {
				return err
			}

			// Find the target agent
			var agentConfig *runtime.AgentConfig
			if agent != "" {
				for i, a := range config.Agents {
					if a.Name == agent {
						agentConfig = &config.Agents[i]
						break
					}
				}
				if agentConfig == nil {
					return fmt.Errorf("agent %q not found", agent)
				}
			} else {
				agentConfig = &config.Agents[0]
			}

			if input == "" {
				return fmt.Errorf("--input is required")
			}

			// Create LLM client
			llmClient := llm.NewAnthropicClient()

			// Create tool registry
			registry := tools.NewRegistry()

			// Create session manager
			sessionStore := session.NewMemoryStore(0)
			memoryStore := memory.NewSlidingWindow(50)
			_ = session.NewManager(sessionStore, memoryStore)

			// Create strategy
			strategy := &loop.ReActStrategy{}

			ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer cancel()

			inv := loop.Invocation{
				AgentName:   agentConfig.Name,
				Model:       agentConfig.Model,
				System:      agentConfig.System,
				Input:       input,
				MaxTurns:    agentConfig.MaxTurns,
				MaxTokens:   4096,
				TokenBudget: agentConfig.TokenBudget,
				Temperature: agentConfig.Temperature,
				Stream:      stream,
			}

			var onEvent loop.StreamCallback
			if stream {
				onEvent = func(event llm.StreamEvent) {
					switch event.Type {
					case "text":
						fmt.Print(event.Text)
					case "tool_call_start":
						if event.ToolCall != nil {
							fmt.Fprintf(os.Stderr, "\n[calling tool: %s]\n", event.ToolCall.Name)
						}
					}
				}
			}

			resp, err := strategy.Execute(ctx, inv, llmClient, registry, onEvent)
			if err != nil {
				return fmt.Errorf("invocation failed: %w", err)
			}

			if stream {
				fmt.Println()
			} else {
				fmt.Println(resp.Output)
			}

			if verbose {
				stats := map[string]interface{}{
					"turns":       resp.Turns,
					"duration_ms": resp.Duration.Milliseconds(),
					"tokens":      resp.Tokens,
					"tool_calls":  len(resp.ToolCalls),
				}
				data, _ := json.MarshalIndent(stats, "", "  ")
				fmt.Fprintf(os.Stderr, "\n%s\n", string(data))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&input, "input", "", "Message to send to the agent")
	cmd.Flags().StringVar(&agent, "agent", "", "Agent name (defaults to first agent)")
	cmd.Flags().BoolVar(&stream, "stream", false, "Stream response")

	return cmd
}
