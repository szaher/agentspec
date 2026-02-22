package ir

import (
	"fmt"
	"strings"
)

// ApplyEnvironment applies environment overrides to the IR resources.
// It takes a base document and an environment name, finds the
// matching environment resource, and applies its overrides.
func ApplyEnvironment(doc *Document, envName string) (*Document, error) {
	if envName == "" {
		return doc, nil
	}

	// Find the environment resource
	var envResource *Resource
	for i := range doc.Resources {
		if doc.Resources[i].Kind == "Environment" && doc.Resources[i].Name == envName {
			envResource = &doc.Resources[i]
			break
		}
	}

	if envResource == nil {
		return nil, fmt.Errorf("environment %q not found", envName)
	}

	// Extract overrides from the environment resource
	overridesRaw, ok := envResource.Attributes["overrides"]
	if !ok {
		return doc, nil
	}

	overrides, ok := overridesRaw.([]interface{})
	if !ok {
		return doc, nil
	}

	// Apply each override to the target resource
	result := *doc
	result.Resources = make([]Resource, len(doc.Resources))
	copy(result.Resources, doc.Resources)

	for _, ovRaw := range overrides {
		ov, ok := ovRaw.(map[string]interface{})
		if !ok {
			continue
		}

		resourceRef, _ := ov["resource"].(string)
		attribute, _ := ov["attribute"].(string)
		value, _ := ov["value"].(string)

		if resourceRef == "" || attribute == "" {
			continue
		}

		// Find and update the target resource
		// Normalize: the parser produces lowercase kinds (e.g., "agent/name")
		// but IR resources have titlecase kinds (e.g., "Agent")
		applied := false
		for i := range result.Resources {
			resourceKey := result.Resources[i].Kind + "/" + result.Resources[i].Name
			if resourceKey == resourceRef || strings.EqualFold(resourceKey, resourceRef) {
				// Copy attributes to avoid modifying original
				newAttrs := make(map[string]interface{})
				for k, v := range result.Resources[i].Attributes {
					newAttrs[k] = v
				}
				newAttrs[attribute] = value
				result.Resources[i].Attributes = newAttrs
				// Recompute hash
				result.Resources[i].Hash = ComputeHash(newAttrs)
				applied = true
				break
			}
		}

		if !applied {
			return nil, fmt.Errorf("environment %q: override target %q not found", envName, resourceRef)
		}
	}

	// Remove environment resources from the output (they're metadata, not deployable)
	var filtered []Resource
	for _, r := range result.Resources {
		if r.Kind != "Environment" {
			filtered = append(filtered, r)
		}
	}
	result.Resources = filtered

	return &result, nil
}
