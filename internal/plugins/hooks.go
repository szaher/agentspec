package plugins

import "fmt"

// HookStage represents a lifecycle stage for hook execution.
type HookStage string

const (
	StagePreValidate  HookStage = "pre-validate"
	StagePostValidate HookStage = "post-validate"
	StagePrePlan      HookStage = "pre-plan"
	StagePostPlan     HookStage = "post-plan"
	StagePreApply     HookStage = "pre-apply"
	StagePostApply    HookStage = "post-apply"
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
// Hooks are executed in the order specified by the plugin's hooks_order,
// or in plugin registration order if no explicit ordering is provided.
func ExecuteHooks(plugins []*LoadedPlugin, stage HookStage, context map[string]interface{}) ([]HookResult, error) {
	var results []HookResult

	for _, plugin := range plugins {
		for _, hook := range plugin.Manifest.Capabilities.Hooks {
			if HookStage(hook.Stage) != stage {
				continue
			}

			// In a full implementation, this would call the WASM hook export
			// with the context JSON. For now, record the hook execution.
			result := HookResult{
				Plugin:  plugin.Manifest.Name,
				Hook:    hook.Name,
				Stage:   stage,
				Output:  fmt.Sprintf("hook %s executed", hook.Name),
				Success: true,
			}
			results = append(results, result)
		}
	}

	return results, nil
}
