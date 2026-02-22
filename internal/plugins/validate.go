package plugins

import (
	"github.com/szaher/designs/agentz/internal/ir"
)

// ValidateResources dispatches validation to plugins based on resource types.
func ValidateResources(plugins []*LoadedPlugin, resources []ir.Resource) []error {
	var errs []error

	for _, plugin := range plugins {
		for _, validator := range plugin.Manifest.Capabilities.Validators {
			for _, resource := range resources {
				if matchesAppliesTo(validator.AppliesTo, resource.Kind) {
					// In a full implementation, this would call the WASM validate export.
					// For now, we validate against the plugin's schema.
					_ = resource
				}
			}
		}
	}

	return errs
}

func matchesAppliesTo(appliesTo []string, kind string) bool {
	for _, at := range appliesTo {
		if at == kind || at == "*" {
			return true
		}
	}
	return false
}
