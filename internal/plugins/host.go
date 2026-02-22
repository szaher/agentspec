// Package plugins implements the WASM plugin host and dispatch
// for the Agentz toolchain.
package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Host manages WASM plugin instances.
type Host struct {
	runtime wazero.Runtime
	plugins map[string]*LoadedPlugin
}

// LoadedPlugin represents a loaded WASM plugin with its manifest.
type LoadedPlugin struct {
	Manifest Manifest
	module   wazero.CompiledModule
}

// NewHost creates a new WASM plugin host.
func NewHost(ctx context.Context) (*Host, error) {
	rt := wazero.NewRuntime(ctx)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	return &Host{
		runtime: rt,
		plugins: make(map[string]*LoadedPlugin),
	}, nil
}

// LoadPlugin loads a WASM plugin from the given path.
func (h *Host) LoadPlugin(ctx context.Context, path string) (*LoadedPlugin, error) {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading plugin %s: %w", path, err)
	}

	compiled, err := h.runtime.CompileModule(ctx, wasmBytes)
	if err != nil {
		return nil, fmt.Errorf("compiling plugin %s: %w", path, err)
	}

	// Instantiate to get manifest
	config := wazero.NewModuleConfig().
		WithStdout(os.Stdout).
		WithStderr(os.Stderr)

	mod, err := h.runtime.InstantiateModule(ctx, compiled, config)
	if err != nil {
		return nil, fmt.Errorf("instantiating plugin %s: %w", path, err)
	}
	defer func() { _ = mod.Close(ctx) }()

	// Call manifest export
	manifestFn := mod.ExportedFunction("manifest")
	if manifestFn == nil {
		return nil, fmt.Errorf("plugin %s does not export 'manifest' function", path)
	}

	results, err := manifestFn.Call(ctx)
	if err != nil {
		return nil, fmt.Errorf("calling manifest in %s: %w", path, err)
	}

	if len(results) < 2 {
		return nil, fmt.Errorf("manifest function returned unexpected results")
	}

	ptr := uint32(results[0])
	size := uint32(results[1])
	mem := mod.Memory()
	data, ok := mem.Read(ptr, size)
	if !ok {
		return nil, fmt.Errorf("reading manifest memory failed")
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	plugin := &LoadedPlugin{
		Manifest: manifest,
		module:   compiled,
	}

	h.plugins[manifest.Name] = plugin
	return plugin, nil
}

// GetPlugin returns a loaded plugin by name.
func (h *Host) GetPlugin(name string) (*LoadedPlugin, bool) {
	p, ok := h.plugins[name]
	return p, ok
}

// Close releases all plugin resources.
func (h *Host) Close(ctx context.Context) error {
	return h.runtime.Close(ctx)
}
