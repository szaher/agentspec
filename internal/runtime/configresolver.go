package runtime

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ConfigParamDef describes a declared config parameter from the IR.
type ConfigParamDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Secret      bool   `json:"secret"`
	HasDefault  bool   `json:"has_default"`
	Default     string `json:"default,omitempty"`
}

// ResolvedConfig holds resolved configuration values for an agent.
type ResolvedConfig struct {
	AgentName string                 `json:"agent_name"`
	Values    map[string]interface{} `json:"values"`
}

// ConfigResolver resolves declared config params from environment
// variables and optional config files. The resolution order is:
//  1. Environment variable AGENTSPEC_<AGENT>_<PARAM>
//  2. Config file (if provided)
//  3. Default value (if declared)
type ConfigResolver struct {
	configFile map[string]interface{} // loaded from --config file
}

// NewConfigResolver creates a new ConfigResolver. If configFilePath
// is non-empty, it loads the config file as JSON.
func NewConfigResolver(configFilePath string) (*ConfigResolver, error) {
	r := &ConfigResolver{}
	if configFilePath != "" {
		data, err := os.ReadFile(configFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading config file %q: %w", configFilePath, err)
		}
		var cfg map[string]interface{}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, fmt.Errorf("parsing config file %q: %w", configFilePath, err)
		}
		r.configFile = cfg
	}
	return r, nil
}

// Resolve resolves all declared config params for an agent.
// Returns an error if any required param is missing.
func (r *ConfigResolver) Resolve(agentName string, params []ConfigParamDef) (*ResolvedConfig, error) {
	resolved := &ResolvedConfig{
		AgentName: agentName,
		Values:    make(map[string]interface{}),
	}

	var missing []string

	for _, p := range params {
		envKey := configEnvKey(agentName, p.Name)
		val, found := r.resolveValue(agentName, p.Name, envKey)

		if found {
			typed, err := convertValue(val, p.Type)
			if err != nil {
				return nil, fmt.Errorf("config param %q (%s): %w", p.Name, envKey, err)
			}
			resolved.Values[p.Name] = typed
			continue
		}

		if p.HasDefault {
			typed, err := convertValue(p.Default, p.Type)
			if err != nil {
				return nil, fmt.Errorf("config param %q default value: %w", p.Name, err)
			}
			resolved.Values[p.Name] = typed
			continue
		}

		if p.Required {
			missing = append(missing, fmt.Sprintf("%s (env: %s)", p.Name, envKey))
		}
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required config params: %s", strings.Join(missing, ", "))
	}

	return resolved, nil
}

// resolveValue checks env var first, then config file.
func (r *ConfigResolver) resolveValue(agentName, paramName, envKey string) (string, bool) {
	// 1. Environment variable (highest priority)
	if val, ok := os.LookupEnv(envKey); ok {
		return val, true
	}

	// 2. Config file
	if r.configFile != nil {
		// Check under agent-specific section
		if agentCfg, ok := r.configFile[agentName]; ok {
			if m, ok := agentCfg.(map[string]interface{}); ok {
				if v, ok := m[paramName]; ok {
					return fmt.Sprintf("%v", v), true
				}
			}
		}
		// Check top-level
		if v, ok := r.configFile[paramName]; ok {
			return fmt.Sprintf("%v", v), true
		}
	}

	return "", false
}

// configEnvKey generates the environment variable name for a config param.
// Pattern: AGENTSPEC_<AGENT>_<PARAM> (uppercased, hyphens to underscores).
func configEnvKey(agentName, paramName string) string {
	agent := strings.ToUpper(strings.ReplaceAll(agentName, "-", "_"))
	param := strings.ToUpper(strings.ReplaceAll(paramName, "-", "_"))
	return "AGENTSPEC_" + agent + "_" + param
}

// convertValue converts a string value to the declared type.
func convertValue(val, typ string) (interface{}, error) {
	switch typ {
	case "string":
		return val, nil
	case "int":
		n, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int: %w", val, err)
		}
		return n, nil
	case "float":
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float: %w", val, err)
		}
		return f, nil
	case "bool":
		b, err := strconv.ParseBool(val)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to bool: %w", val, err)
		}
		return b, nil
	default:
		return val, nil // treat unknown types as string
	}
}

// ExtractConfigParams extracts ConfigParamDef from IR agent attributes.
func ExtractConfigParams(attrs map[string]interface{}) []ConfigParamDef {
	raw, ok := attrs["config_params"].([]interface{})
	if !ok {
		return nil
	}

	var params []ConfigParamDef
	for _, item := range raw {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		p := ConfigParamDef{
			Name: strVal(m, "name"),
			Type: strVal(m, "type"),
		}
		if d, ok := m["description"].(string); ok {
			p.Description = d
		}
		if r, ok := m["required"].(bool); ok {
			p.Required = r
		}
		if s, ok := m["secret"].(bool); ok {
			p.Secret = s
		}
		if d, ok := m["default"]; ok {
			p.HasDefault = true
			p.Default = fmt.Sprintf("%v", d)
		}
		params = append(params, p)
	}
	return params
}

func strVal(m map[string]interface{}, key string) string {
	v, _ := m[key].(string)
	return v
}
