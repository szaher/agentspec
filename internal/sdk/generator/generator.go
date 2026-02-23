// Package generator implements the type codegen engine for SDK generation.
package generator

import (
	"fmt"
	"os"
	"path/filepath"
)

// Language represents a target SDK language.
type Language string

const (
	LangPython     Language = "python"
	LangTypeScript Language = "typescript"
	LangGo         Language = "go"
)

// Config holds SDK generation configuration.
type Config struct {
	Language Language
	OutDir   string
	IRSchema map[string]interface{}
}

// Generate produces SDK files for the configured language.
func Generate(cfg Config) error {
	switch cfg.Language {
	case LangPython:
		return generatePython(cfg)
	case LangTypeScript:
		return generateTypeScript(cfg)
	case LangGo:
		return generateGo(cfg)
	default:
		return fmt.Errorf("unsupported language: %s", cfg.Language)
	}
}

func generatePython(cfg Config) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return err
	}

	// Generate __init__.py
	initContent := `"""AgentSpec SDK for Python."""
from .client import AgentSpecClient, AsyncAgentSpecClient
from .types import Agent, Prompt, Skill, MCPServer, MCPClient, ResourceSummary
from .errors import ResourceNotFoundError, StateFileError, InvocationError

__all__ = [
    "AgentSpecClient",
    "AsyncAgentSpecClient",
    "Agent",
    "Prompt",
    "Skill",
    "MCPServer",
    "MCPClient",
    "ResourceSummary",
    "ResourceNotFoundError",
    "StateFileError",
    "InvocationError",
]
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "__init__.py"), []byte(initContent), 0644); err != nil {
		return err
	}

	// Generate types.py
	typesContent := `"""Generated types from IR schema."""
from dataclasses import dataclass
from typing import Any, Dict, List, Optional

@dataclass
class ResourceSummary:
    name: str
    kind: str
    fqn: str
    status: str
    hash: str
    last_applied: str

@dataclass
class Agent:
    name: str
    fqn: str
    model: str
    prompt: str
    skills: List[str]
    status: str
    hash: str
    attributes: Dict[str, Any]

@dataclass
class Prompt:
    name: str
    fqn: str
    content: str
    status: str
    hash: str
    attributes: Dict[str, Any]

@dataclass
class Skill:
    name: str
    fqn: str
    description: str
    status: str
    hash: str
    attributes: Dict[str, Any]

@dataclass
class MCPServer:
    name: str
    fqn: str
    transport: str
    status: str
    hash: str
    attributes: Dict[str, Any]

@dataclass
class MCPClient:
    name: str
    fqn: str
    servers: List[str]
    status: str
    hash: str
    attributes: Dict[str, Any]
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "types.py"), []byte(typesContent), 0644); err != nil {
		return err
	}

	// Generate errors.py
	errorsContent := `"""SDK error types."""

class AgentSpecError(Exception):
    pass

class ResourceNotFoundError(AgentSpecError):
    def __init__(self, kind: str, name: str):
        super().__init__(f"{kind} '{name}' not found")
        self.kind = kind
        self.name = name

class StateFileError(AgentSpecError):
    def __init__(self, path: str, message: str):
        super().__init__(f"State file error ({path}): {message}")
        self.path = path

class InvocationError(AgentSpecError):
    def __init__(self, agent: str, message: str):
        super().__init__(f"Invocation error for '{agent}': {message}")
        self.agent = agent
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "errors.py"), []byte(errorsContent), 0644); err != nil {
		return err
	}

	// Generate client.py
	clientContent := `"""AgentSpec SDK client."""
import json
from pathlib import Path
from typing import List, Optional
from .types import Agent, Prompt, Skill, MCPServer, MCPClient, ResourceSummary
from .errors import ResourceNotFoundError, StateFileError

class AgentSpecClient:
    def __init__(self, state_file: str = ".agentspec.state.json"):
        self._state_file = Path(state_file)
        self._state = self._load_state()

    def _load_state(self) -> dict:
        try:
            with open(self._state_file) as f:
                return json.load(f)
        except FileNotFoundError:
            raise StateFileError(str(self._state_file), "file not found")
        except json.JSONDecodeError as e:
            raise StateFileError(str(self._state_file), str(e))

    def _get_entries(self, kind: str) -> list:
        return [e for e in self._state.get("entries", [])
                if f"/{kind}/" in e.get("fqn", "")]

    def list_agents(self) -> List[ResourceSummary]:
        return [ResourceSummary(
            name=e["fqn"].split("/")[-1],
            kind="Agent",
            fqn=e["fqn"],
            status=e.get("status", "unknown"),
            hash=e.get("hash", ""),
            last_applied=e.get("last_applied", ""),
        ) for e in self._get_entries("Agent")]

    def list_prompts(self) -> List[ResourceSummary]:
        return [ResourceSummary(
            name=e["fqn"].split("/")[-1],
            kind="Prompt",
            fqn=e["fqn"],
            status=e.get("status", "unknown"),
            hash=e.get("hash", ""),
            last_applied=e.get("last_applied", ""),
        ) for e in self._get_entries("Prompt")]

    def list_skills(self) -> List[ResourceSummary]:
        return [ResourceSummary(
            name=e["fqn"].split("/")[-1],
            kind="Skill",
            fqn=e["fqn"],
            status=e.get("status", "unknown"),
            hash=e.get("hash", ""),
            last_applied=e.get("last_applied", ""),
        ) for e in self._get_entries("Skill")]

    def list_servers(self) -> List[ResourceSummary]:
        return [ResourceSummary(
            name=e["fqn"].split("/")[-1],
            kind="MCPServer",
            fqn=e["fqn"],
            status=e.get("status", "unknown"),
            hash=e.get("hash", ""),
            last_applied=e.get("last_applied", ""),
        ) for e in self._get_entries("MCPServer")]

    def list_clients(self) -> List[ResourceSummary]:
        return [ResourceSummary(
            name=e["fqn"].split("/")[-1],
            kind="MCPClient",
            fqn=e["fqn"],
            status=e.get("status", "unknown"),
            hash=e.get("hash", ""),
            last_applied=e.get("last_applied", ""),
        ) for e in self._get_entries("MCPClient")]

    def get_agent(self, name: str) -> Optional[ResourceSummary]:
        for a in self.list_agents():
            if a.name == name:
                return a
        raise ResourceNotFoundError("Agent", name)


class AsyncAgentSpecClient:
    """Async version of AgentSpecClient."""
    def __init__(self, state_file: str = ".agentspec.state.json"):
        self._sync = AgentSpecClient(state_file)

    async def list_agents(self) -> List[ResourceSummary]:
        return self._sync.list_agents()

    async def get_agent(self, name: str) -> Optional[ResourceSummary]:
        return self._sync.get_agent(name)
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "client.py"), []byte(clientContent), 0644); err != nil {
		return err
	}

	// Generate setup.py
	setupContent := `from setuptools import setup, find_packages

setup(
    name="agentspec",
    version="0.1.0",
    packages=find_packages(),
    python_requires=">=3.10",
)
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "setup.py"), []byte(setupContent), 0644); err != nil {
		return err
	}

	return nil
}

func generateTypeScript(cfg Config) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return err
	}

	// Generate index.ts
	indexContent := `// @agentspec/sdk - Generated SDK
export { AgentSpecClient } from './client';
export type { Agent, Prompt, Skill, MCPServer, MCPClient, ResourceSummary } from './types';
export { ResourceNotFoundError, StateFileError, InvocationError } from './errors';
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "index.ts"), []byte(indexContent), 0644); err != nil {
		return err
	}

	// Generate types.ts
	typesContent := `// Generated types from IR schema
export interface ResourceSummary {
  name: string;
  kind: string;
  fqn: string;
  status: 'applied' | 'failed' | 'pending';
  hash: string;
  lastApplied: string;
}

export interface Agent extends ResourceSummary {
  model: string;
  prompt: string;
  skills: string[];
  attributes: Record<string, unknown>;
}

export interface Prompt extends ResourceSummary {
  content: string;
  attributes: Record<string, unknown>;
}

export interface Skill extends ResourceSummary {
  description: string;
  attributes: Record<string, unknown>;
}

export interface MCPServer extends ResourceSummary {
  transport: 'stdio' | 'sse' | 'streamable-http';
  attributes: Record<string, unknown>;
}

export interface MCPClient extends ResourceSummary {
  servers: string[];
  attributes: Record<string, unknown>;
}
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "types.ts"), []byte(typesContent), 0644); err != nil {
		return err
	}

	// Generate errors.ts
	errorsContent := "export class AgentSpecError extends Error {\n  constructor(message: string) {\n    super(message);\n    this.name = 'AgentSpecError';\n  }\n}\n\nexport class ResourceNotFoundError extends AgentSpecError {\n  constructor(public readonly kind: string, public readonly resourceName: string) {\n    super(`${kind} '${resourceName}' not found`);\n    this.name = 'ResourceNotFoundError';\n  }\n}\n\nexport class StateFileError extends AgentSpecError {\n  constructor(public readonly path: string, reason: string) {\n    super(`State file error (${path}): ${reason}`);\n    this.name = 'StateFileError';\n  }\n}\n\nexport class InvocationError extends AgentSpecError {\n  constructor(public readonly agent: string, reason: string) {\n    super(`Invocation error for '${agent}': ${reason}`);\n    this.name = 'InvocationError';\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "errors.ts"), []byte(errorsContent), 0644); err != nil {
		return err
	}

	// Generate client.ts
	clientContent := "import * as fs from 'fs';\nimport type { ResourceSummary } from './types';\nimport { ResourceNotFoundError, StateFileError } from './errors';\n\ninterface StateFile {\n  version: string;\n  entries: StateEntry[];\n}\n\ninterface StateEntry {\n  fqn: string;\n  hash: string;\n  status: string;\n  last_applied: string;\n  adapter: string;\n  error?: string;\n}\n\nexport class AgentSpecClient {\n  private state: StateFile;\n\n  constructor(private stateFile: string = '.agentspec.state.json') {\n    this.state = this.loadState();\n  }\n\n  private loadState(): StateFile {\n    try {\n      const data = fs.readFileSync(this.stateFile, 'utf-8');\n      return JSON.parse(data);\n    } catch (err: any) {\n      throw new StateFileError(this.stateFile, err.message);\n    }\n  }\n\n  private getEntries(kind: string): StateEntry[] {\n    return this.state.entries.filter(e => e.fqn.includes(`/${kind}/`));\n  }\n\n  async listAgents(): Promise<ResourceSummary[]> {\n    return this.getEntries('Agent').map(e => ({\n      name: e.fqn.split('/').pop()!,\n      kind: 'Agent',\n      fqn: e.fqn,\n      status: e.status as any,\n      hash: e.hash,\n      lastApplied: e.last_applied,\n    }));\n  }\n\n  async getAgent(name: string): Promise<ResourceSummary> {\n    const agents = await this.listAgents();\n    const agent = agents.find(a => a.name === name);\n    if (!agent) throw new ResourceNotFoundError('Agent', name);\n    return agent;\n  }\n}\n"
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "client.ts"), []byte(clientContent), 0644); err != nil {
		return err
	}

	// Generate package.json
	pkgContent := `{
  "name": "@agentspec/sdk",
  "version": "0.1.0",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  "scripts": {
    "build": "tsc"
  }
}
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "package.json"), []byte(pkgContent), 0644); err != nil {
		return err
	}

	return nil
}

func generateGo(cfg Config) error {
	if err := os.MkdirAll(cfg.OutDir, 0755); err != nil {
		return err
	}

	// Generate client.go
	clientContent := `// Package agentspec provides a Go SDK for accessing AgentSpec resources.
package agentspec

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Client provides access to AgentSpec resources via the state file.
type Client struct {
	stateFile string
	state     *stateData
}

type stateData struct {
	Version string       ` + "`" + `json:"version"` + "`" + `
	Entries []StateEntry ` + "`" + `json:"entries"` + "`" + `
}

// StateEntry represents a resource in the state file.
type StateEntry struct {
	FQN         string ` + "`" + `json:"fqn"` + "`" + `
	Hash        string ` + "`" + `json:"hash"` + "`" + `
	Status      string ` + "`" + `json:"status"` + "`" + `
	LastApplied string ` + "`" + `json:"last_applied"` + "`" + `
	Adapter     string ` + "`" + `json:"adapter"` + "`" + `
	Error       string ` + "`" + `json:"error,omitempty"` + "`" + `
}

// ResourceSummary provides a summary of a resource.
type ResourceSummary struct {
	Name        string
	Kind        string
	FQN         string
	Status      string
	Hash        string
	LastApplied string
}

// NewClient creates a new AgentSpec client.
func NewClient(stateFile string) (*Client, error) {
	c := &Client{stateFile: stateFile}
	if err := c.loadState(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Client) loadState() error {
	data, err := os.ReadFile(c.stateFile)
	if err != nil {
		return fmt.Errorf("state file error (%s): %w", c.stateFile, err)
	}
	c.state = &stateData{}
	return json.Unmarshal(data, c.state)
}

func (c *Client) getEntries(kind string) []StateEntry {
	var result []StateEntry
	for _, e := range c.state.Entries {
		if strings.Contains(e.FQN, "/"+kind+"/") {
			result = append(result, e)
		}
	}
	return result
}

// ListAgents returns all agent resources.
func (c *Client) ListAgents(ctx context.Context) []ResourceSummary {
	return c.toSummaries("Agent", c.getEntries("Agent"))
}

// GetAgent returns a specific agent by name.
func (c *Client) GetAgent(ctx context.Context, name string) (*ResourceSummary, error) {
	for _, a := range c.ListAgents(ctx) {
		if a.Name == name {
			return &a, nil
		}
	}
	return nil, fmt.Errorf("agent %q not found", name)
}

func (c *Client) toSummaries(kind string, entries []StateEntry) []ResourceSummary {
	result := make([]ResourceSummary, 0, len(entries))
	for _, e := range entries {
		parts := strings.Split(e.FQN, "/")
		name := parts[len(parts)-1]
		result = append(result, ResourceSummary{
			Name:        name,
			Kind:        kind,
			FQN:         e.FQN,
			Status:      e.Status,
			Hash:        e.Hash,
			LastApplied: e.LastApplied,
		})
	}
	return result
}
`
	if err := os.WriteFile(filepath.Join(cfg.OutDir, "client.go"), []byte(clientContent), 0644); err != nil {
		return err
	}

	return nil
}
