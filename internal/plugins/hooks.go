package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
)

// HookStage represents a lifecycle stage for hook execution.
type HookStage string

const (
	StagePreValidate  HookStage = "pre-validate"
	StagePostValidate HookStage = "post-validate"
	StagePrePlan      HookStage = "pre-plan"
	StagePostPlan     HookStage = "post-plan"
	StagePreApply     HookStage = "pre-apply"
	StagePostApply    HookStage = "post-apply"
	StagePreInvoke    HookStage = "pre-invoke"
	StagePostInvoke   HookStage = "post-invoke"
	StageRuntime      HookStage = "runtime"
)

// HookResult contains the output of a hook execution.
type HookResult struct {
	Plugin  string
	Hook    string
	Stage   HookStage
	Output  string
	Success bool
	Error   string
}

// ExecuteHooks runs all registered hooks for a given stage.
func ExecuteHooks(host *Host, plugins []*LoadedPlugin, stage HookStage, hookCtx map[string]interface{}) ([]HookResult, error) {
	var results []HookResult

	for _, plugin := range plugins {
		for _, hook := range plugin.Manifest.Capabilities.Hooks {
			if HookStage(hook.Stage) != stage {
				continue
			}

			result := callWASMHook(host, plugin, hook, stage, hookCtx)
			results = append(results, result)
		}
	}

	return results, nil
}

func callWASMHook(host *Host, plugin *LoadedPlugin, hook Hook, stage HookStage, hookCtx map[string]interface{}) HookResult {
	result := HookResult{
		Plugin: plugin.Manifest.Name,
		Hook:   hook.Name,
		Stage:  stage,
	}

	if plugin.module == nil {
		result.Output = fmt.Sprintf("hook %s: no compiled module, skipping", hook.Name)
		result.Success = true
		return result
	}

	ctx := context.Background()

	// Serialize context to JSON
	ctxData, err := json.Marshal(hookCtx)
	if err != nil {
		result.Error = fmt.Sprintf("marshal context: %v", err)
		return result
	}

	// Instantiate the module
	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr).
		WithName(fmt.Sprintf("%s-%s-%s", plugin.Manifest.Name, hook.Name, string(stage)))

	mod, err := host.runtime.InstantiateModule(ctx, plugin.module, config)
	if err != nil {
		result.Error = fmt.Sprintf("instantiate: %v", err)
		return result
	}
	defer func() { _ = mod.Close(ctx) }()

	// Find the hook export function
	hookFnName := fmt.Sprintf("hook_%s", hook.Name)
	hookFn := mod.ExportedFunction(hookFnName)
	if hookFn == nil {
		// Try the stage-based naming convention
		hookFn = mod.ExportedFunction(string(stage))
	}
	if hookFn == nil {
		result.Output = fmt.Sprintf("hook %s: no export found, skipping", hook.Name)
		result.Success = true
		return result
	}

	// Write context data to WASM memory
	allocFn := mod.ExportedFunction("alloc")
	if allocFn == nil {
		result.Error = "plugin does not export 'alloc' function"
		return result
	}

	allocResults, err := allocFn.Call(ctx, uint64(len(ctxData)))
	if err != nil {
		result.Error = fmt.Sprintf("alloc: %v", err)
		return result
	}

	ptr := uint32(allocResults[0])
	if !mod.Memory().Write(ptr, ctxData) {
		result.Error = "failed to write context to WASM memory"
		return result
	}

	// Call the hook
	hookResults, err := hookFn.Call(ctx, uint64(ptr), uint64(len(ctxData)))
	if err != nil {
		result.Error = fmt.Sprintf("hook call: %v", err)
		return result
	}

	// Read result from memory
	if len(hookResults) >= 2 {
		resultPtr := uint32(hookResults[0])
		resultSize := uint32(hookResults[1])
		if resultData, ok := mod.Memory().Read(resultPtr, resultSize); ok {
			result.Output = string(resultData)
		}
	}

	result.Success = true
	return result
}
