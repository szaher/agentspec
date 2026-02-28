package compiler

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
)

// DetectedFeature represents an AgentSpec feature found in the IR document.
type DetectedFeature struct {
	Name       string // canonical feature name
	ResourceFQN string // which resource uses this feature
}

// GapWarning represents a feature gap for a specific target.
type GapWarning struct {
	Feature    string
	Level      plugins.FeatureSupportLevel
	Message    string
	Suggestion string
}

// DetectFeatures inspects an IR document and returns all AgentSpec features in use.
func DetectFeatures(doc *ir.Document) []DetectedFeature {
	var features []DetectedFeature

	for _, r := range doc.Resources {
		switch r.Kind {
		case "Agent":
			features = append(features, DetectedFeature{Name: "agent", ResourceFQN: r.FQN})
			features = append(features, detectAgentFeatures(r)...)
		case "Skill":
			features = append(features, DetectedFeature{Name: "skill", ResourceFQN: r.FQN})
			features = append(features, detectSkillFeatures(r)...)
		case "Prompt":
			features = append(features, DetectedFeature{Name: "prompt", ResourceFQN: r.FQN})
			if vars, ok := r.Attributes["variables"]; ok && vars != nil {
				features = append(features, DetectedFeature{Name: "prompt_variables", ResourceFQN: r.FQN})
			}
		case "Pipeline":
			features = append(features, DetectedFeature{Name: "pipeline_sequential", ResourceFQN: r.FQN})
			features = append(features, detectPipelineFeatures(r)...)
		case "Type":
			features = append(features, DetectedFeature{Name: "type_definitions", ResourceFQN: r.FQN})
		}
	}

	return features
}

func detectAgentFeatures(r ir.Resource) []DetectedFeature {
	var features []DetectedFeature

	if strategy, ok := r.Attributes["strategy"].(string); ok {
		switch strategy {
		case "react":
			features = append(features, DetectedFeature{Name: "loop_react", ResourceFQN: r.FQN})
		case "plan-and-execute", "plan_and_execute":
			features = append(features, DetectedFeature{Name: "loop_plan_execute", ResourceFQN: r.FQN})
		case "reflexion":
			features = append(features, DetectedFeature{Name: "loop_reflexion", ResourceFQN: r.FQN})
		case "router":
			features = append(features, DetectedFeature{Name: "loop_router", ResourceFQN: r.FQN})
		case "map-reduce", "map_reduce":
			features = append(features, DetectedFeature{Name: "loop_map_reduce", ResourceFQN: r.FQN})
		}
	}

	if cp, ok := r.Attributes["config_params"]; ok && cp != nil {
		features = append(features, DetectedFeature{Name: "config_params", ResourceFQN: r.FQN})
	}
	if vr, ok := r.Attributes["validation_rules"]; ok && vr != nil {
		features = append(features, DetectedFeature{Name: "validation_rules", ResourceFQN: r.FQN})
	}
	if ec, ok := r.Attributes["eval_cases"]; ok && ec != nil {
		features = append(features, DetectedFeature{Name: "eval_cases", ResourceFQN: r.FQN})
	}
	if oi, ok := r.Attributes["on_input"]; ok && oi != nil {
		features = append(features, detectControlFlowFeatures(r, oi)...)
	}
	if d, ok := r.Attributes["delegates"]; ok && d != nil {
		features = append(features, DetectedFeature{Name: "delegation", ResourceFQN: r.FQN})
	}
	if s, ok := r.Attributes["stream"]; ok && s == true {
		features = append(features, DetectedFeature{Name: "streaming", ResourceFQN: r.FQN})
	}
	if m, ok := r.Attributes["memory"]; ok && m != nil {
		features = append(features, DetectedFeature{Name: "sessions", ResourceFQN: r.FQN})
	}
	if f, ok := r.Attributes["fallback"]; ok && f != nil && f != "" {
		features = append(features, DetectedFeature{Name: "error_handling", ResourceFQN: r.FQN})
	}

	return features
}

func detectSkillFeatures(r ir.Resource) []DetectedFeature {
	var features []DetectedFeature

	if tool, ok := r.Attributes["tool"].(map[string]interface{}); ok {
		if toolType, ok := tool["type"].(string); ok {
			switch toolType {
			case "inline":
				features = append(features, DetectedFeature{Name: "inline_tools", ResourceFQN: r.FQN})
			case "mcp":
				features = append(features, DetectedFeature{Name: "mcp_tools", ResourceFQN: r.FQN})
			}
		}
	}

	return features
}

func detectPipelineFeatures(r ir.Resource) []DetectedFeature {
	var features []DetectedFeature

	if steps, ok := r.Attributes["steps"].([]interface{}); ok {
		for _, step := range steps {
			if s, ok := step.(map[string]interface{}); ok {
				if when, ok := s["when"].(string); ok && when != "" {
					features = append(features, DetectedFeature{Name: "pipeline_conditional", ResourceFQN: r.FQN})
				}
				if parallel, ok := s["parallel"].(bool); ok && parallel {
					features = append(features, DetectedFeature{Name: "pipeline_hierarchical", ResourceFQN: r.FQN})
				}
			}
		}
	}

	return features
}

func detectControlFlowFeatures(r ir.Resource, onInput interface{}) []DetectedFeature {
	var features []DetectedFeature
	stmts, ok := onInput.([]interface{})
	if !ok {
		return features
	}

	for _, stmt := range stmts {
		m, ok := stmt.(map[string]interface{})
		if !ok {
			continue
		}
		stmtType, _ := m["type"].(string)
		switch stmtType {
		case "if":
			features = append(features, DetectedFeature{Name: "control_flow_if", ResourceFQN: r.FQN})
		case "for_each":
			features = append(features, DetectedFeature{Name: "control_flow_foreach", ResourceFQN: r.FQN})
		}
	}

	return features
}

// AnalyzeGaps compares detected features against a target's feature support map
// and returns warnings for features that are not fully supported.
func AnalyzeGaps(features []DetectedFeature, featureMap plugins.FeatureMap) []GapWarning {
	// Deduplicate features
	seen := make(map[string]string) // feature name -> first resource FQN
	for _, f := range features {
		if _, ok := seen[f.Name]; !ok {
			seen[f.Name] = f.ResourceFQN
		}
	}

	var warnings []GapWarning
	for featureName, resourceFQN := range seen {
		level, exists := featureMap[featureName]
		if !exists {
			level = plugins.FeatureNone
		}

		switch level {
		case plugins.FeatureFull:
			// No warning needed
		case plugins.FeaturePartial:
			warnings = append(warnings, GapWarning{
				Feature:    featureName,
				Level:      level,
				Message:    fmt.Sprintf("Feature %q (used by %s) has partial support in this target", featureName, resourceFQN),
				Suggestion: gapSuggestion(featureName, level),
			})
		case plugins.FeatureEmulated:
			warnings = append(warnings, GapWarning{
				Feature:    featureName,
				Level:      level,
				Message:    fmt.Sprintf("Feature %q (used by %s) is emulated via generated code, not a native framework primitive", featureName, resourceFQN),
				Suggestion: gapSuggestion(featureName, level),
			})
		case plugins.FeatureNone:
			warnings = append(warnings, GapWarning{
				Feature:    featureName,
				Level:      level,
				Message:    fmt.Sprintf("Feature %q (used by %s) is not supported by this target", featureName, resourceFQN),
				Suggestion: gapSuggestion(featureName, level),
			})
		}
	}

	return warnings
}

func gapSuggestion(feature string, level plugins.FeatureSupportLevel) string {
	if level == plugins.FeatureNone {
		suggestions := map[string]string{
			"loop_reflexion":       "Consider using the LangGraph target which supports reflexion loops",
			"loop_router":         "Consider using the LangGraph target which supports conditional routing",
			"loop_map_reduce":     "Consider using the LangGraph target for map-reduce patterns",
			"pipeline_conditional": "Consider using the LangGraph target for conditional pipelines",
			"inline_tools":        "Convert inline tools to command or HTTP tools for this target",
			"mcp_tools":           "Convert MCP tools to HTTP or command tools for this target",
			"control_flow_if":     "Consider using the standalone or LangGraph target for control flow",
			"control_flow_foreach": "Consider using the standalone or LangGraph target for loops",
		}
		if s, ok := suggestions[feature]; ok {
			return s
		}
		return "This feature will be omitted from the generated code"
	}
	if level == plugins.FeatureEmulated {
		return "Generated as application-level code wrapper; review the implementation for correctness"
	}
	return "Check target documentation for limitations"
}

// GapWarningsToStrings converts gap warnings to simple string messages for the CompileResult.
func GapWarningsToStrings(warnings []GapWarning) []string {
	var result []string
	for _, w := range warnings {
		msg := fmt.Sprintf("[%s] %s", w.Level, w.Message)
		if w.Suggestion != "" {
			msg += " — " + w.Suggestion
		}
		result = append(result, msg)
	}
	return result
}
