package plugins

import (
	"github.com/szaher/designs/agentz/internal/ir"
)

// TransformResources dispatches transforms to plugins at compile stage.
func TransformResources(plugins []*LoadedPlugin, resources []ir.Resource) ([]ir.Resource, error) {
	result := make([]ir.Resource, len(resources))
	copy(result, resources)

	for _, plugin := range plugins {
		for _, transform := range plugin.Manifest.Capabilities.Transforms {
			if transform.Stage != "compile" {
				continue
			}
			for i := range result {
				if result[i].Kind == transform.InputKind {
					// In a full implementation, this would call the WASM transform export.
					// The transform can modify or expand resources.
					_ = transform
				}
			}
		}
	}

	return result, nil
}
