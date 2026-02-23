package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"

	"github.com/szaher/designs/agentz/internal/ir"
)

// TransformResources dispatches transforms to plugins at compile stage.
func TransformResources(host *Host, plugins []*LoadedPlugin, resources []ir.Resource) ([]ir.Resource, error) {
	result := make([]ir.Resource, len(resources))
	copy(result, resources)

	for _, plugin := range plugins {
		for _, transform := range plugin.Manifest.Capabilities.Transforms {
			if transform.Stage != "compile" {
				continue
			}
			for i := range result {
				if result[i].Kind == transform.InputKind {
					transformed, err := callWASMTransform(host, plugin, transform, result[i])
					if err != nil {
						return nil, err
					}
					result[i] = transformed
				}
			}
		}
	}

	return result, nil
}

func callWASMTransform(host *Host, plugin *LoadedPlugin, transform Transform, resource ir.Resource) (ir.Resource, error) {
	if plugin.module == nil {
		return resource, nil // No compiled module, return unchanged
	}

	ctx := context.Background()

	// Serialize resource to JSON
	resData, err := json.Marshal(resource)
	if err != nil {
		return resource, fmt.Errorf("plugin %s: marshal resource: %w", plugin.Manifest.Name, err)
	}

	// Instantiate the module
	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithName(fmt.Sprintf("%s-transform-%s", plugin.Manifest.Name, resource.Name))

	mod, err := host.runtime.InstantiateModule(ctx, plugin.module, config)
	if err != nil {
		return resource, fmt.Errorf("plugin %s: instantiate: %w", plugin.Manifest.Name, err)
	}
	defer func() { _ = mod.Close(ctx) }()

	// Find transform export
	transformFn := mod.ExportedFunction(fmt.Sprintf("transform_%s", transform.Name))
	if transformFn == nil {
		transformFn = mod.ExportedFunction("transform")
	}
	if transformFn == nil {
		return resource, nil // No export, return unchanged
	}

	// Allocate and write resource data
	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		return resource, fmt.Errorf("plugin %s: no alloc export", plugin.Manifest.Name)
	}

	allocResults, err := allocFn.Call(ctx, uint64(len(resData)))
	if err != nil {
		return resource, fmt.Errorf("plugin %s: alloc: %w", plugin.Manifest.Name, err)
	}

	ptr := uint32(allocResults[0])
	if !mod.Memory().Write(ptr, resData) {
		return resource, fmt.Errorf("plugin %s: write memory failed", plugin.Manifest.Name)
	}

	// Call transform
	results, err := transformFn.Call(ctx, uint64(ptr), uint64(len(resData)))
	if err != nil {
		return resource, fmt.Errorf("plugin %s: transform call: %w", plugin.Manifest.Name, err)
	}

	// Read transformed resource
	if len(results) >= 2 {
		resultPtr := uint32(results[0])
		resultSize := uint32(results[1])
		if resultSize > 0 {
			if resultData, ok := mod.Memory().Read(resultPtr, resultSize); ok {
				var transformed ir.Resource
				if err := json.Unmarshal(resultData, &transformed); err != nil {
					return resource, fmt.Errorf("plugin %s: unmarshal transform result: %w", plugin.Manifest.Name, err)
				}
				return transformed, nil
			}
		}
	}

	return resource, nil
}
