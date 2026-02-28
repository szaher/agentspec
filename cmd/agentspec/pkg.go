package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/szaher/designs/agentz/internal/adapters/docker"
	"github.com/szaher/designs/agentz/internal/adapters/kubernetes"
	"github.com/szaher/designs/agentz/internal/compiler"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
)

// PackageResult is the output of the package command.
type PackageResult struct {
	Status    string   `json:"status"`
	Format    string   `json:"format"`
	OutputDir string   `json:"output_dir"`
	Tag       string   `json:"tag,omitempty"`
	Files     []string `json:"files"`
}

func newPackageCmd() *cobra.Command {
	var (
		format     string
		outputDir  string
		tag        string
		registry   string
		platform   string
		push       bool
		jsonOutput bool
	)

	cmd := &cobra.Command{
		Use:   "package [file.ias | directory | compiled-binary]",
		Short: "Package a compiled agent for deployment",
		Long: `Package a compiled agent binary or .ias files into deployment-ready
artifacts: Docker images, Kubernetes manifests, or Helm charts.

If given .ias files, the package command will compile them first
using the standalone target, then package the resulting binary.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]

			// Determine if input is a compiled binary or .ias file(s)
			var binaryPath string
			var doc *ir.Document

			info, err := os.Stat(input)
			if err != nil {
				return fmt.Errorf("cannot access %q: %w", input, err)
			}

			if !info.IsDir() && filepath.Ext(input) != ".ias" {
				// Assume it's a compiled binary
				binaryPath = input
			} else {
				// Compile .ias files first
				files, err := resolveCompileInputs(args)
				if err != nil {
					return err
				}

				// For container formats, cross-compile for linux
				compilePlatform := platform
				if compilePlatform == "" {
					switch format {
					case "docker", "kubernetes", "k8s", "helm":
						compilePlatform = "linux/amd64"
					}
				}

				if verbose {
					fmt.Fprintf(os.Stderr, "Compiling %d file(s) before packaging...\n", len(files))
					if compilePlatform != "" {
						fmt.Fprintf(os.Stderr, "Target platform: %s\n", compilePlatform)
					}
				}

				compileResult, err := compiler.Compile(files, compiler.CompileOptions{
					Target:    "standalone",
					OutputDir: outputDir,
					Platform:  compilePlatform,
					Verbose:   verbose,
				})
				if err != nil {
					return fmt.Errorf("compilation failed: %w", err)
				}
				binaryPath = compileResult.OutputPath

				// Parse for IR document (needed for K8s/Compose)
				content, err := os.ReadFile(files[0])
				if err != nil {
					return fmt.Errorf("reading %s: %w", files[0], err)
				}
				f, parseErrs := parser.Parse(string(content), files[0])
				if parseErrs != nil {
					return fmt.Errorf("parse errors: %v", parseErrs)
				}
				doc, err = ir.Lower(f)
				if err != nil {
					return fmt.Errorf("lowering to IR: %w", err)
				}
			}

			if outputDir == "" {
				outputDir = "./build"
			}

			if tag == "" {
				tag = filepath.Base(binaryPath) + ":latest"
			}

			var result *PackageResult

			switch format {
			case "docker":
				result, err = packageDocker(binaryPath, outputDir, tag, registry, push)
			case "kubernetes", "k8s":
				result, err = packageKubernetes(binaryPath, doc, outputDir, tag)
			case "helm":
				result, err = packageHelm(binaryPath, doc, outputDir, tag)
			case "binary":
				// Just copy the binary
				result = &PackageResult{
					Status:    "success",
					Format:    "binary",
					OutputDir: filepath.Dir(binaryPath),
					Files:     []string{binaryPath},
				}
			default:
				return fmt.Errorf("unsupported format: %q (available: docker, kubernetes, helm, binary)", format)
			}

			if err != nil {
				return err
			}

			if jsonOutput {
				out, _ := json.MarshalIndent(result, "", "  ")
				fmt.Println(string(out))
			} else {
				printPackageResult(result)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "docker", "Package format: docker, kubernetes, helm, binary")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "./build", "Output directory")
	cmd.Flags().StringVar(&tag, "tag", "", "Image tag (for docker format)")
	cmd.Flags().StringVar(&registry, "registry", "", "Container registry URL")
	cmd.Flags().StringVar(&platform, "platform", "", "Target platform (e.g. linux/amd64, linux/arm64). Auto-detected for container formats")
	cmd.Flags().BoolVar(&push, "push", false, "Push image to registry after build")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output JSON result")

	return cmd
}

func packageDocker(binaryPath, outputDir, tag, registry string, push bool) (*PackageResult, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	binaryName := filepath.Base(binaryPath)

	// Copy the compiled binary into the output directory if it's not already there
	destBinary := filepath.Join(outputDir, binaryName)
	absBinary, _ := filepath.Abs(binaryPath)
	absDest, _ := filepath.Abs(destBinary)
	if absBinary != absDest {
		data, err := os.ReadFile(binaryPath)
		if err != nil {
			return nil, fmt.Errorf("reading binary %s: %w", binaryPath, err)
		}
		if err := os.WriteFile(destBinary, data, 0755); err != nil {
			return nil, fmt.Errorf("copying binary to output dir: %w", err)
		}
	}

	dockerfile := docker.GenerateDockerfileFromBinary(binaryPath, 8080)
	dockerfilePath := filepath.Join(outputDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0644); err != nil {
		return nil, fmt.Errorf("writing Dockerfile: %w", err)
	}

	// Write .dockerignore
	dockerignorePath := filepath.Join(outputDir, ".dockerignore")
	dockerignore := "*.config.md\n.dockerignore\n"
	if err := os.WriteFile(dockerignorePath, []byte(dockerignore), 0644); err != nil {
		return nil, fmt.Errorf("writing .dockerignore: %w", err)
	}

	files := []string{destBinary, dockerfilePath, dockerignorePath}

	result := &PackageResult{
		Status:    "success",
		Format:    "docker",
		OutputDir: outputDir,
		Tag:       tag,
		Files:     files,
	}

	if registry != "" {
		result.Tag = registry + "/" + tag
	}

	return result, nil
}

func packageKubernetes(binaryPath string, doc *ir.Document, outputDir, tag string) (*PackageResult, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, err
	}

	var resources []ir.Resource
	if doc != nil {
		resources = doc.Resources
	}

	config := map[string]interface{}{
		"image": tag,
		"port":  8080,
	}

	manifests := kubernetes.GenerateManifests(resources, config)
	if err := kubernetes.WriteManifests(manifests, outputDir); err != nil {
		return nil, fmt.Errorf("writing manifests: %w", err)
	}

	var files []string
	entries, _ := os.ReadDir(outputDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			files = append(files, filepath.Join(outputDir, e.Name()))
		}
	}

	return &PackageResult{
		Status:    "success",
		Format:    "kubernetes",
		OutputDir: outputDir,
		Tag:       tag,
		Files:     files,
	}, nil
}

func packageHelm(binaryPath string, doc *ir.Document, outputDir, tag string) (*PackageResult, error) {
	chartDir := filepath.Join(outputDir, "chart")

	name := filepath.Base(binaryPath)
	if ext := filepath.Ext(name); ext != "" {
		name = name[:len(name)-len(ext)]
	}

	chart := kubernetes.GenerateHelmChart(name, tag, 8080)
	if err := kubernetes.WriteHelmChart(chart, chartDir); err != nil {
		return nil, fmt.Errorf("writing Helm chart: %w", err)
	}

	var files []string
	filepath.Walk(chartDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return &PackageResult{
		Status:    "success",
		Format:    "helm",
		OutputDir: chartDir,
		Tag:       tag,
		Files:     files,
	}, nil
}

func printPackageResult(result *PackageResult) {
	fmt.Printf("Package format: %s\n", result.Format)
	fmt.Printf("Output: %s\n", result.OutputDir)
	if result.Tag != "" {
		fmt.Printf("Tag: %s\n", result.Tag)
	}
	fmt.Printf("Files generated: %d\n", len(result.Files))
	for _, f := range result.Files {
		fmt.Printf("  %s\n", f)
	}

	// Print next steps
	fmt.Println()
	switch result.Format {
	case "docker":
		fmt.Println("Next steps:")
		fmt.Printf("  1. Build the image:  docker build -t %s %s\n", result.Tag, result.OutputDir)
		fmt.Printf("  2. Run the container: docker run -p 8080:8080 %s\n", result.Tag)
		fmt.Printf("  3. Open the UI:       http://localhost:8080\n")
	case "kubernetes":
		fmt.Println("Next steps:")
		fmt.Printf("  1. Build and push a Docker image for your agent binary\n")
		fmt.Printf("  2. Apply manifests:  kubectl apply -f %s/\n", result.OutputDir)
	case "helm":
		fmt.Println("Next steps:")
		fmt.Printf("  1. Build and push a Docker image for your agent binary\n")
		fmt.Printf("  2. Install chart:    helm install my-agent %s\n", result.OutputDir)
	case "binary":
		fmt.Println("Next steps:")
		fmt.Printf("  1. Run the binary:   %s --port 8080\n", result.Files[0])
		fmt.Printf("  2. Open the UI:      http://localhost:8080\n")
	}
}
