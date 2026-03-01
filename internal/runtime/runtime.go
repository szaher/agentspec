package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/szaher/designs/agentz/internal/llm"
	"github.com/szaher/designs/agentz/internal/loop"
	agentmcp "github.com/szaher/designs/agentz/internal/mcp"
	"github.com/szaher/designs/agentz/internal/memory"
	"github.com/szaher/designs/agentz/internal/secrets"
	"github.com/szaher/designs/agentz/internal/session"
	"github.com/szaher/designs/agentz/internal/tools"
)

// Runtime manages the full lifecycle of an agent runtime.
type Runtime struct {
	config   *RuntimeConfig
	server   *Server
	mcpPool  *agentmcp.Pool
	registry *tools.Registry
	logger   *slog.Logger
	apiKey   string
	port     int
}

// Options configures the runtime.
type Options struct {
	Port        int
	APIKey      string
	NoAuth      bool
	CORSOrigins []string
	Logger      *slog.Logger
	LLMClient   llm.Client
	EnableUI    bool
}

// New creates a new runtime from the given config.
func New(config *RuntimeConfig, opts Options) (*Runtime, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	port := opts.Port
	if port == 0 {
		port = 8080
	}

	// Create LLM client — auto-detect provider from first agent's model string
	llmClient := opts.LLMClient
	if llmClient == nil {
		if len(config.Agents) > 0 {
			var resolvedModel string
			llmClient, resolvedModel = llm.NewClientForModel(config.Agents[0].Model)
			// Update agent model to the resolved name (without provider prefix)
			for i := range config.Agents {
				_, m := llm.ParseModelString(config.Agents[i].Model)
				config.Agents[i].Model = m
			}
			_ = resolvedModel
		} else {
			llmClient = llm.NewAnthropicClient()
		}
	}

	// Create MCP connection pool
	mcpPool := agentmcp.NewPool()

	// Create tool registry
	registry := tools.NewRegistry()

	// Create secret resolver
	resolver := secrets.NewEnvResolver()

	// Create session manager
	sessionStore := session.NewMemoryStore(30 * time.Minute)
	memoryStore := memory.NewSlidingWindow(50)
	sessionMgr := session.NewManager(sessionStore, memoryStore)

	// Create strategy
	strategy := &loop.ReActStrategy{}

	// Create server
	var serverOpts []ServerOption
	if opts.APIKey != "" {
		serverOpts = append(serverOpts, WithAPIKey(opts.APIKey))
	} else if opts.NoAuth {
		logger.Warn("server starting WITHOUT authentication (--no-auth flag provided)")
	} else {
		logger.Warn("no API key configured: all API requests will be rejected. Use --no-auth to explicitly allow unauthenticated access, or set AGENTSPEC_API_KEY")
	}
	serverOpts = append(serverOpts, WithLogger(logger))
	serverOpts = append(serverOpts, WithNoAuth(opts.NoAuth))
	if len(opts.CORSOrigins) > 0 {
		serverOpts = append(serverOpts, WithCORSOrigins(opts.CORSOrigins))
	}
	if opts.EnableUI {
		serverOpts = append(serverOpts, WithUI(true))
	}

	server := NewServer(config, llmClient, registry, sessionMgr, strategy, serverOpts...)

	rt := &Runtime{
		config:   config,
		server:   server,
		mcpPool:  mcpPool,
		registry: registry,
		logger:   logger,
		apiKey:   opts.APIKey,
		port:     port,
	}

	// Register tools
	if err := rt.registerTools(context.Background(), resolver); err != nil {
		return nil, fmt.Errorf("register tools: %w", err)
	}

	return rt, nil
}

// Start starts the runtime (MCP servers, HTTP server).
func (rt *Runtime) Start(ctx context.Context) error {
	// Start MCP servers
	for _, srv := range rt.config.MCPServers {
		rt.logger.Info("starting MCP server", "name", srv.Name, "command", srv.Command)
		_, err := rt.mcpPool.Connect(ctx, agentmcp.ServerConfig{
			Name:      srv.Name,
			Transport: srv.Transport,
			Command:   srv.Command,
			Args:      srv.Args,
		})
		if err != nil {
			return fmt.Errorf("start MCP server %s: %w", srv.Name, err)
		}
	}

	// Discover MCP tools and register them
	discovery := agentmcp.NewDiscovery(rt.mcpPool)
	mcpTools, err := discovery.DiscoverTools(ctx)
	if err != nil {
		rt.logger.Warn("MCP tool discovery failed", "error", err)
	}

	for _, t := range mcpTools {
		rt.logger.Info("discovered MCP tool", "server", t.ServerName, "tool", t.Name)
		toolName := t.ServerName + "/" + t.Name
		client, _ := rt.mcpPool.Get(t.ServerName)
		rt.registry.Register(toolName, llm.ToolDefinition{
			Name:        toolName,
			Description: t.Description,
			InputSchema: t.InputSchema,
		}, &mcpToolExecutor{client: client, toolName: t.Name})
	}

	// Start HTTP server
	addr := fmt.Sprintf(":%d", rt.port)
	return rt.server.ListenAndServe(addr)
}

// Shutdown gracefully stops the runtime.
func (rt *Runtime) Shutdown(ctx context.Context) error {
	rt.logger.Info("shutting down runtime")

	if err := rt.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	if err := rt.mcpPool.Close(); err != nil {
		return fmt.Errorf("close MCP pool: %w", err)
	}

	return nil
}

// Port returns the configured port.
func (rt *Runtime) Port() int {
	return rt.port
}

func (rt *Runtime) registerTools(ctx context.Context, resolver secrets.Resolver) error {
	for _, skill := range rt.config.Skills {
		if skill.Tool == nil {
			continue
		}

		toolType, _ := skill.Tool["type"].(string)
		switch toolType {
		case "mcp":
			// MCP tools are registered via discovery during Start
			continue
		case "http":
			config := tools.HTTPConfig{
				Method: getStr(skill.Tool, "method"),
				URL:    getStr(skill.Tool, "url"),
			}
			if headers, ok := skill.Tool["headers"].(map[string]interface{}); ok {
				config.Headers = make(map[string]string)
				for k, v := range headers {
					config.Headers[k], _ = v.(string)
				}
			}
			config.BodyTemplate, _ = skill.Tool["body_template"].(string)
			rt.registry.Register(skill.Name, llm.ToolDefinition{
				Name:        skill.Name,
				Description: skill.Description,
			}, tools.NewHTTPExecutor(config))

		case "command":
			config := tools.CommandConfig{
				Binary: getStr(skill.Tool, "binary"),
			}
			if args, ok := skill.Tool["args"].([]interface{}); ok {
				for _, a := range args {
					if s, ok := a.(string); ok {
						config.Args = append(config.Args, s)
					}
				}
			}
			// Resolve secrets
			resolvedSecrets := make(map[string]string)
			if secs, ok := skill.Tool["secrets"].(map[string]interface{}); ok {
				for k, v := range secs {
					if ref, ok := v.(string); ok {
						val, err := resolver.Resolve(ctx, ref)
						if err != nil {
							rt.logger.Warn("secret resolution failed", "key", k, "error", err)
							continue
						}
						resolvedSecrets[k] = val
					}
				}
			}
			rt.registry.Register(skill.Name, llm.ToolDefinition{
				Name:        skill.Name,
				Description: skill.Description,
			}, tools.NewCommandExecutor(config, resolvedSecrets))

		case "inline":
			config := tools.InlineConfig{
				Language: getStr(skill.Tool, "language"),
				Code:     getStr(skill.Tool, "code"),
			}
			resolvedSecrets := make(map[string]string)
			if secs, ok := skill.Tool["secrets"].(map[string]interface{}); ok {
				for k, v := range secs {
					if ref, ok := v.(string); ok {
						val, err := resolver.Resolve(ctx, ref)
						if err != nil {
							rt.logger.Warn("secret resolution failed", "key", k, "error", err)
							continue
						}
						resolvedSecrets[k] = val
					}
				}
			}
			rt.registry.Register(skill.Name, llm.ToolDefinition{
				Name:        skill.Name,
				Description: skill.Description,
			}, tools.NewInlineExecutor(config, resolvedSecrets))
		}
	}
	return nil
}

func getStr(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}

// mcpToolExecutor executes tools via MCP.
type mcpToolExecutor struct {
	client   *agentmcp.Client
	toolName string
}

func (e *mcpToolExecutor) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	return e.client.CallTool(ctx, e.toolName, input)
}
