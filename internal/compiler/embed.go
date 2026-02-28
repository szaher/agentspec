package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/szaher/designs/agentz/internal/runtime"
)

// StandaloneResult is the output of a standalone compilation.
type StandaloneResult struct {
	OutputPath  string   `json:"output_path"`
	SizeBytes   int64    `json:"size_bytes"`
	ContentHash string   `json:"content_hash"`
	Platform    string   `json:"platform"`
	Agents      []string `json:"agents"`
	ConfigRef   string   `json:"config_ref"`
	Duration    time.Duration
}

// CompileStandalone compiles a RuntimeConfig into a standalone Go binary.
// It creates a temporary build directory INSIDE the module tree (under
// cmd/_build/) so the generated main.go can import internal packages,
// then invokes go build.
func CompileStandalone(config *runtime.RuntimeConfig, opts CompileOptions) (*StandaloneResult, error) {
	startTime := time.Now()

	root := moduleRoot()

	// Create temp build directory inside module tree
	buildDir := filepath.Join(root, "cmd", "_build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("creating build dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(buildDir) }()

	// Serialize config to JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serializing config: %w", err)
	}

	// Write config.json
	configPath := filepath.Join(buildDir, "config.json")
	if err := os.WriteFile(configPath, configJSON, 0644); err != nil {
		return nil, fmt.Errorf("writing config.json: %w", err)
	}

	// Generate main.go from template
	mainSrc := generateMain(TemplateData{
		Version:   opts.Version,
		BuildTime: time.Now().UTC().Format(time.RFC3339),
		Target:    "standalone",
	})

	mainPath := filepath.Join(buildDir, "main.go")
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0644); err != nil {
		return nil, fmt.Errorf("writing main.go: %w", err)
	}

	// Ensure output directory exists
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = "./build"
	}
	if !filepath.IsAbs(outputDir) {
		cwd, _ := os.Getwd()
		outputDir = filepath.Join(cwd, outputDir)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	// Determine output binary name
	binaryName := opts.Name
	if binaryName == "" {
		binaryName = config.PackageName
	}

	// Add .exe extension for Windows targets
	if strings.HasPrefix(opts.Platform, "windows/") {
		binaryName += ".exe"
	}

	outputPath := filepath.Join(outputDir, binaryName)

	// Build the binary from within the module root
	buildArgs := []string{
		"build",
		"-trimpath",
		"-ldflags=-s -w",
		"-o", outputPath,
		"./cmd/_build",
	}

	cmd := exec.Command("go", buildArgs...)
	cmd.Dir = root
	cmd.Env = buildEnv(opts.Platform)

	if opts.Verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go build failed: %w\n%s", err, string(output))
	}

	// Get binary info
	info, err := os.Stat(outputPath)
	if err != nil {
		return nil, fmt.Errorf("stat output binary: %w", err)
	}

	// Compute content hash
	binaryData, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("reading output binary: %w", err)
	}
	hash := sha256.Sum256(binaryData)
	contentHash := "sha256:" + hex.EncodeToString(hash[:])

	// Collect agent names
	var agents []string
	for _, a := range config.Agents {
		agents = append(agents, a.Name)
	}

	// Generate config reference
	var agentRefs []AgentConfigRef
	for _, a := range config.Agents {
		agentRefs = append(agentRefs, AgentConfigRef{
			AgentName: a.Name,
			Params:    a.ConfigParams,
		})
	}

	configRefPath := filepath.Join(outputDir, binaryName+".config.md")
	configRefContent := GenerateConfigRef(agentRefs, binaryName)
	if err := os.WriteFile(configRefPath, []byte(configRefContent), 0644); err != nil {
		return nil, fmt.Errorf("writing config reference: %w", err)
	}

	return &StandaloneResult{
		OutputPath:  outputPath,
		SizeBytes:   info.Size(),
		ContentHash: contentHash,
		Platform:    opts.Platform,
		Agents:      agents,
		ConfigRef:   configRefPath,
		Duration:    time.Since(startTime),
	}, nil
}

func generateMain(data TemplateData) string {
	src := mainTemplate
	src = strings.ReplaceAll(src, "{{.Version}}", data.Version)
	src = strings.ReplaceAll(src, "{{.BuildTime}}", data.BuildTime)
	src = strings.ReplaceAll(src, "{{.Target}}", data.Target)
	return src
}

// moduleRoot finds the go module root by walking up from CWD.
func moduleRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "." // fallback
		}
		dir = parent
	}
}

func buildEnv(platform string) []string {
	env := os.Environ()

	if platform == "" {
		return env
	}

	parts := strings.SplitN(platform, "/", 2)
	if len(parts) != 2 {
		return env
	}

	// Filter out existing GOOS/GOARCH
	var filtered []string
	for _, e := range env {
		if !strings.HasPrefix(e, "GOOS=") && !strings.HasPrefix(e, "GOARCH=") {
			filtered = append(filtered, e)
		}
	}

	filtered = append(filtered, "GOOS="+parts[0], "GOARCH="+parts[1])

	// Disable CGO for cross-compilation
	hasCGO := false
	for _, e := range filtered {
		if strings.HasPrefix(e, "CGO_ENABLED=") {
			hasCGO = true
			break
		}
	}
	if !hasCGO {
		filtered = append(filtered, "CGO_ENABLED=0")
	}

	return filtered
}
