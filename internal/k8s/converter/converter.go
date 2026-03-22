// Package converter maps IntentLang IR resources to Kubernetes CRD manifests.
package converter

import (
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1alpha1 "github.com/szaher/agentspec/internal/api/v1alpha1"
	"github.com/szaher/agentspec/internal/ir"
)

// Resource represents a generated Kubernetes resource.
type Resource struct {
	APIVersion string
	Kind       string
	Name       string
	Namespace  string
	Raw        map[string]interface{}
}

// ConvertDocument maps an IR Document to Kubernetes CRD resources.
func ConvertDocument(doc *ir.Document, namespace string) ([]Resource, error) {
	var resources []Resource

	for _, res := range doc.Resources {
		converted, err := convertResource(res, namespace)
		if err != nil {
			return nil, fmt.Errorf("converting %s/%s: %w", res.Kind, res.Name, err)
		}
		resources = append(resources, converted...)
	}

	// Policies from the IR use a different format (allow/deny rules) than the
	// CRD's structured PolicySpec (cost budgets, rate limits, content filters).
	// Skip automatic conversion; users should create Policy CRs directly.

	return resources, nil
}

func convertResource(res ir.Resource, namespace string) ([]Resource, error) {
	switch res.Kind {
	case "Agent":
		return convertAgent(res, namespace)
	case "Prompt":
		return convertPrompt(res, namespace)
	case "Skill":
		return convertSkill(res, namespace)
	case "Secret":
		return convertSecret(res, namespace)
	case "Pipeline":
		return convertPipeline(res, namespace)
	case "MCPServer":
		return convertMCPServer(res, namespace)
	default:
		// Skip unsupported kinds (Type, Environment, etc.)
		return nil, nil
	}
}

func convertAgent(res ir.Resource, namespace string) ([]Resource, error) {
	agent := &v1alpha1.Agent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "agentspec.io/v1alpha1",
			Kind:       "Agent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      res.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.AgentSpec{
			Model: stringAttr(res.Attributes, "model"),
		},
	}

	if v := stringAttr(res.Attributes, "strategy"); v != "" {
		agent.Spec.Strategy = v
	}
	if v := stringAttr(res.Attributes, "prompt"); v != "" {
		agent.Spec.PromptRef = v
	}
	if v := intAttr(res.Attributes, "max_turns"); v > 0 {
		agent.Spec.MaxTurns = int32(v)
	}

	// Map skills to ToolBinding refs.
	if skills, ok := res.Attributes["skills"].([]interface{}); ok {
		for _, s := range skills {
			if name, ok := s.(string); ok {
				agent.Spec.SkillRefs = append(agent.Spec.SkillRefs, name)
			}
		}
	}

	raw, err := toRawMap(agent)
	if err != nil {
		return nil, err
	}

	return []Resource{{
		APIVersion: "agentspec.io/v1alpha1",
		Kind:       "Agent",
		Name:       res.Name,
		Namespace:  namespace,

		Raw: raw,
	}}, nil
}

func convertPrompt(res ir.Resource, namespace string) ([]Resource, error) {
	content := stringAttr(res.Attributes, "content")
	raw := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      res.Name,
			"namespace": namespace,
		},
		"data": map[string]interface{}{
			"system-prompt": content,
		},
	}

	return []Resource{{
		APIVersion: "v1",
		Kind:       "ConfigMap",
		Name:       res.Name,
		Namespace:  namespace,
		Raw:        raw,
	}}, nil
}

func convertSkill(res ir.Resource, namespace string) ([]Resource, error) {
	tb := &v1alpha1.ToolBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "agentspec.io/v1alpha1",
			Kind:       "ToolBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      res.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ToolBindingSpec{
			Name:        res.Name,
			Description: stringAttr(res.Attributes, "description"),
		},
	}

	// Determine tool type from the tool config.
	if tool, ok := res.Attributes["tool"].(map[string]interface{}); ok {
		toolType := stringFromMap(tool, "type")

		switch toolType {
		case "inline":
			// Inline bash/shell tools map to "command" with sh -c.
			tb.Spec.ToolType = "command"
			code := stringFromMap(tool, "code")
			tb.Spec.Command = &v1alpha1.CommandToolSpec{
				Binary: "sh",
				Args:   []string{"-c", code},
			}
		case "mcp":
			tb.Spec.ToolType = "mcp"
			tb.Spec.MCP = &v1alpha1.MCPToolSpec{
				ServerRef: stringFromMap(tool, "server_tool"),
			}
		case "command":
			tb.Spec.ToolType = "command"
			tb.Spec.Command = &v1alpha1.CommandToolSpec{
				Binary: stringFromMap(tool, "binary"),
			}
			if args, ok := tool["args"].([]interface{}); ok {
				for _, a := range args {
					if s, ok := a.(string); ok {
						tb.Spec.Command.Args = append(tb.Spec.Command.Args, s)
					}
				}
			}
		case "http":
			tb.Spec.ToolType = "http"
			tb.Spec.HTTP = &v1alpha1.HTTPToolSpec{
				URL:    stringFromMap(tool, "url"),
				Method: stringFromMap(tool, "method"),
			}
		default:
			tb.Spec.ToolType = toolType
		}
	}

	raw, err := toRawMap(tb)
	if err != nil {
		return nil, err
	}

	return []Resource{{
		APIVersion: "agentspec.io/v1alpha1",
		Kind:       "ToolBinding",
		Name:       res.Name,
		Namespace:  namespace,

		Raw: raw,
	}}, nil
}

func convertSecret(res ir.Resource, namespace string) ([]Resource, error) {
	raw := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]interface{}{
			"name":      res.Name,
			"namespace": namespace,
		},
		"type": "Opaque",
		"stringData": map[string]interface{}{
			"key": stringAttr(res.Attributes, "key"),
		},
	}

	return []Resource{{
		APIVersion: "v1",
		Kind:       "Secret",
		Name:       res.Name,
		Namespace:  namespace,
		Raw:        raw,
	}}, nil
}

func convertPipeline(res ir.Resource, namespace string) ([]Resource, error) {
	var steps []v1alpha1.WorkflowStep

	if stepsRaw, ok := res.Attributes["steps"].([]interface{}); ok {
		for _, s := range stepsRaw {
			stepMap, ok := s.(map[string]interface{})
			if !ok {
				continue
			}
			step := v1alpha1.WorkflowStep{
				Name:     stringFromMap(stepMap, "name"),
				AgentRef: stringFromMap(stepMap, "agent"),
				Input:    stringFromMap(stepMap, "input"),
			}
			if deps, ok := stepMap["depends_on"].([]interface{}); ok {
				for _, d := range deps {
					if name, ok := d.(string); ok {
						step.DependsOn = append(step.DependsOn, name)
					}
				}
			}
			steps = append(steps, step)
		}
	}

	wf := &v1alpha1.Workflow{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "agentspec.io/v1alpha1",
			Kind:       "Workflow",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      res.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.WorkflowSpec{
			Steps: steps,
		},
	}

	raw, err := toRawMap(wf)
	if err != nil {
		return nil, err
	}

	return []Resource{{
		APIVersion: "agentspec.io/v1alpha1",
		Kind:       "Workflow",
		Name:       res.Name,
		Namespace:  namespace,

		Raw: raw,
	}}, nil
}

func convertMCPServer(res ir.Resource, namespace string) ([]Resource, error) {
	tb := &v1alpha1.ToolBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "agentspec.io/v1alpha1",
			Kind:       "ToolBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      res.Name,
			Namespace: namespace,
		},
		Spec: v1alpha1.ToolBindingSpec{
			Name:     res.Name,
			ToolType: "mcp",
			MCP: &v1alpha1.MCPToolSpec{
				ServerRef: res.Name,
			},
		},
	}

	raw, err := toRawMap(tb)
	if err != nil {
		return nil, err
	}

	return []Resource{{
		APIVersion: "agentspec.io/v1alpha1",
		Kind:       "ToolBinding",
		Name:       res.Name,
		Namespace:  namespace,

		Raw: raw,
	}}, nil
}

// Helper functions

func stringAttr(attrs map[string]interface{}, key string) string {
	if v, ok := attrs[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func intAttr(attrs map[string]interface{}, key string) int {
	if v, ok := attrs[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return 0
}

func stringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func toRawMap(obj interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}
