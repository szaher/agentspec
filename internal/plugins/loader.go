package plugins

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manifest describes a plugin's capabilities.
type Manifest struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Description  string       `json:"description"`
	Capabilities Capabilities `json:"capabilities"`
	WASM         WASMConfig   `json:"wasm"`
}

// Capabilities lists what the plugin provides.
type Capabilities struct {
	ResourceTypes []ResourceType     `json:"resource_types,omitempty"`
	Validators    []Validator        `json:"validators,omitempty"`
	Transforms    []Transform        `json:"transforms,omitempty"`
	Hooks         []Hook             `json:"hooks,omitempty"`
	Compile       *CompileCapability `json:"compile,omitempty"`
}

// CompileCapability declares that a plugin can compile AgentSpec IR to a framework target.
type CompileCapability struct {
	TargetName          string   `json:"target_name"`
	OutputType          string   `json:"output_type"`
	OutputLanguage      string   `json:"output_language"`
	SupportedFeatures   []string `json:"supported_features,omitempty"`
	UnsupportedFeatures []string `json:"unsupported_features,omitempty"`
}

// ResourceType declares a custom resource kind.
type ResourceType struct {
	Kind   string                 `json:"kind"`
	Schema map[string]interface{} `json:"schema,omitempty"`
}

// Validator declares a custom validator.
type Validator struct {
	Name        string   `json:"name"`
	AppliesTo   []string `json:"applies_to"`
	Description string   `json:"description"`
}

// Transform declares a custom transform.
type Transform struct {
	Name        string `json:"name"`
	Stage       string `json:"stage"`
	InputKind   string `json:"input_kind"`
	Description string `json:"description"`
}

// Hook declares a lifecycle hook.
type Hook struct {
	Stage       string `json:"stage"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// WASMConfig specifies WASM runtime configuration.
type WASMConfig struct {
	MinMemoryPages int      `json:"min_memory_pages"`
	MaxMemoryPages int      `json:"max_memory_pages"`
	Capabilities   []string `json:"capabilities"`
}

// ResolvePluginPath finds a plugin WASM module by name and version.
// Search order: ./plugins/<name>/plugin.wasm, ~/.agentspec/plugins/<name>/<version>/plugin.wasm,
// then fallback to ~/.agentz/plugins/<name>/<version>/plugin.wasm with deprecation warning.
func ResolvePluginPath(name, version string) (string, error) {
	// Local path
	localPath := filepath.Join("plugins", name, "plugin.wasm")
	if _, err := os.Stat(localPath); err == nil {
		return localPath, nil
	}

	home, err := os.UserHomeDir()
	if err == nil {
		// Primary: ~/.agentspec/plugins/
		cachePath := filepath.Join(home, ".agentspec", "plugins", name, version, "plugin.wasm")
		if _, err := os.Stat(cachePath); err == nil {
			return cachePath, nil
		}

		// Fallback: ~/.agentz/plugins/ (deprecated)
		oldCachePath := filepath.Join(home, ".agentz", "plugins", name, version, "plugin.wasm")
		if _, err := os.Stat(oldCachePath); err == nil {
			fmt.Fprintf(os.Stderr,
				"Warning: Plugin '%s' found in deprecated path '~/.agentz/plugins/'. "+
					"Move plugins to '~/.agentspec/plugins/' instead.\n", name)
			return oldCachePath, nil
		}
	}

	return "", fmt.Errorf("plugin %q version %q not found", name, version)
}

// LoadManifestFromFile loads a plugin manifest from a JSON file.
func LoadManifestFromFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// CheckConflicts checks for duplicate resource types and compile targets across plugins.
func CheckConflicts(plugins []*LoadedPlugin) error {
	kinds := make(map[string]string)   // kind -> plugin name
	targets := make(map[string]string) // target_name -> plugin name
	for _, p := range plugins {
		for _, rt := range p.Manifest.Capabilities.ResourceTypes {
			if existing, ok := kinds[rt.Kind]; ok {
				return fmt.Errorf("resource type %q declared by both %q and %q",
					rt.Kind, existing, p.Manifest.Name)
			}
			kinds[rt.Kind] = p.Manifest.Name
		}
		if p.Manifest.Capabilities.Compile != nil {
			target := p.Manifest.Capabilities.Compile.TargetName
			if existing, ok := targets[target]; ok {
				return fmt.Errorf("compile target %q declared by both %q and %q",
					target, existing, p.Manifest.Name)
			}
			targets[target] = p.Manifest.Name
		}
	}
	return nil
}

// FindCompilePlugin returns the plugin that provides a given compilation target.
func FindCompilePlugin(plugins []*LoadedPlugin, targetName string) *LoadedPlugin {
	for _, p := range plugins {
		if p.Manifest.Capabilities.Compile != nil &&
			p.Manifest.Capabilities.Compile.TargetName == targetName {
			return p
		}
	}
	return nil
}

// ListCompileTargets returns all available compilation targets from loaded plugins.
func ListCompileTargets(plugins []*LoadedPlugin) []string {
	var targets []string
	for _, p := range plugins {
		if p.Manifest.Capabilities.Compile != nil {
			targets = append(targets, p.Manifest.Capabilities.Compile.TargetName)
		}
	}
	return targets
}
