package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"

	"github.com/szaher/designs/agentz/internal/ir"
)

// ValidateResources dispatches validation to plugins based on resource types.
func ValidateResources(host *Host, plugins []*LoadedPlugin, resources []ir.Resource) []error {
	var errs []error

	for _, plugin := range plugins {
		for _, validator := range plugin.Manifest.Capabilities.Validators {
			for _, resource := range resources {
				if matchesAppliesTo(validator.AppliesTo, resource.Kind) {
					if err := callWASMValidator(host, plugin, validator, resource); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}
	}

	return errs
}

func callWASMValidator(host *Host, plugin *LoadedPlugin, validator Validator, resource ir.Resource) error {
	if plugin.module == nil {
		return nil // No compiled module, skip
	}

	ctx := context.Background()

	// Serialize resource to JSON
	resData, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("plugin %s: marshal resource: %w", plugin.Manifest.Name, err)
	}

	// Instantiate the module
	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithName(fmt.Sprintf("%s-validate-%s", plugin.Manifest.Name, resource.Name))

	mod, err := host.runtime.InstantiateModule(ctx, plugin.module, config)
	if err != nil {
		return fmt.Errorf("plugin %s: instantiate: %w", plugin.Manifest.Name, err)
	}
	defer func() { _ = mod.Close(ctx) }()

	// Find validator export
	validateFn := mod.ExportedFunction(fmt.Sprintf("validate_%s", validator.Name))
	if validateFn == nil {
		validateFn = mod.ExportedFunction("validate")
	}
	if validateFn == nil {
		return nil // No export, skip
	}

	// Allocate and write resource data
	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		return fmt.Errorf("plugin %s: no alloc export", plugin.Manifest.Name)
	}

	allocResults, err := allocFn.Call(ctx, uint64(len(resData)))
	if err != nil {
		return fmt.Errorf("plugin %s: alloc: %w", plugin.Manifest.Name, err)
	}

	ptr := uint32(allocResults[0])
	if !mod.Memory().Write(ptr, resData) {
		return fmt.Errorf("plugin %s: write memory failed", plugin.Manifest.Name)
	}

	// Call validator
	results, err := validateFn.Call(ctx, uint64(ptr), uint64(len(resData)))
	if err != nil {
		return fmt.Errorf("plugin %s: validate call: %w", plugin.Manifest.Name, err)
	}

	// Read errors from result
	if len(results) >= 2 {
		errPtr := uint32(results[0])
		errSize := uint32(results[1])
		if errSize > 0 {
			if errData, ok := mod.Memory().Read(errPtr, errSize); ok {
				var validationErrors []string
				if json.Unmarshal(errData, &validationErrors) == nil {
					for _, e := range validationErrors {
						return fmt.Errorf("plugin %s: %s: %s", plugin.Manifest.Name, validator.Name, e)
					}
				}
			}
		}
	}

	return nil
}

func matchesAppliesTo(appliesTo []string, kind string) bool {
	for _, at := range appliesTo {
		if at == kind || at == "*" {
			return true
		}
	}
	return false
}
