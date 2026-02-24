package integration_tests

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/validate"
)

// TestDocExamples validates all .ias example files in docs/examples/.
func TestDocExamples(t *testing.T) {
	docsExamplesDir := filepath.Join(getRepoRoot(t), "docs", "examples")

	entries, err := os.ReadDir(docsExamplesDir)
	if err != nil {
		t.Fatalf("failed to read docs/examples directory: %v", err)
	}

	var iasFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".ias") {
			iasFiles = append(iasFiles, e.Name())
		}
	}

	if len(iasFiles) == 0 {
		t.Fatal("no .ias files found in docs/examples/")
	}

	for _, name := range iasFiles {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(docsExamplesDir, name)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read %s: %v", name, err)
			}

			input := string(data)

			// Parse
			f, parseErrs := parser.Parse(input, name)
			if parseErrs != nil {
				t.Fatalf("parse errors in %s: %v", name, parseErrs)
			}
			if f == nil {
				t.Fatalf("parsed file is nil for %s", name)
			}

			// Structural validation
			structErrs := validate.ValidateStructural(f)
			if len(structErrs) > 0 {
				t.Fatalf("structural validation errors in %s: %v", name, structErrs)
			}

			// Semantic validation
			semErrs := validate.ValidateSemantic(f)
			if len(semErrs) > 0 {
				t.Fatalf("semantic validation errors in %s: %v", name, semErrs)
			}
		})
	}
}

// TestDocMarkdownCodeBlocks extracts and validates fenced .ias code blocks
// from all Markdown files in the docs/ directory.
//
// Fence tag conventions:
//   - ```ias           — complete, valid file; must parse and validate
//   - ```ias fragment  — fragment without package header; wrapped with synthetic header
//   - ```ias invalid   — intentionally invalid; must produce a parse or validation error
//   - ```ias novalidate — pseudocode or conceptual; skipped entirely
func TestDocMarkdownCodeBlocks(t *testing.T) {
	docsDir := filepath.Join(getRepoRoot(t), "docs")

	var mdFiles []string
	err := filepath.Walk(docsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".md") {
			mdFiles = append(mdFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to walk docs directory: %v", err)
	}

	for _, mdFile := range mdFiles {
		relPath, _ := filepath.Rel(docsDir, mdFile)
		t.Run(relPath, func(t *testing.T) {
			blocks := extractIASBlocks(t, mdFile)
			for i, block := range blocks {
				t.Run(fmt.Sprintf("block-%d-%s", i+1, block.tag), func(t *testing.T) {
					switch block.tag {
					case "ias":
						validateCompleteBlock(t, block.content, mdFile, i+1)
					case "ias fragment":
						validateFragmentBlock(t, block.content, mdFile, i+1)
					case "ias invalid":
						validateInvalidBlock(t, block.content, mdFile, i+1)
					case "ias novalidate":
						t.Skip("novalidate block — skipped")
					default:
						t.Skipf("unknown ias tag %q — skipped", block.tag)
					}
				})
			}
		})
	}
}

type iasBlock struct {
	tag     string
	content string
	line    int
}

func extractIASBlocks(t *testing.T, mdPath string) []iasBlock {
	t.Helper()

	f, err := os.Open(mdPath)
	if err != nil {
		t.Fatalf("failed to open %s: %v", mdPath, err)
	}
	defer func() { _ = f.Close() }()

	var blocks []iasBlock
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	lineNum := 0
	inBlock := false
	var currentTag string
	var currentContent strings.Builder
	var blockStartLine int

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !inBlock {
			if strings.HasPrefix(trimmed, "```ias") {
				tag := strings.TrimPrefix(trimmed, "```")
				tag = strings.TrimSpace(tag)
				// Only match ias tags, not e.g. "iasm" or other
				if tag == "ias" || tag == "ias fragment" || tag == "ias invalid" || tag == "ias novalidate" {
					inBlock = true
					currentTag = tag
					currentContent.Reset()
					blockStartLine = lineNum
				}
			}
		} else {
			if trimmed == "```" {
				blocks = append(blocks, iasBlock{
					tag:     currentTag,
					content: currentContent.String(),
					line:    blockStartLine,
				})
				inBlock = false
			} else {
				currentContent.WriteString(line)
				currentContent.WriteByte('\n')
			}
		}
	}

	if err := scanner.Err(); err != nil {
		t.Fatalf("error scanning %s: %v", mdPath, err)
	}

	return blocks
}

func validateCompleteBlock(t *testing.T, content, mdFile string, blockNum int) {
	t.Helper()

	f, parseErrs := parser.Parse(content, fmt.Sprintf("%s:block-%d", filepath.Base(mdFile), blockNum))
	if parseErrs != nil {
		t.Fatalf("parse errors in block %d: %v", blockNum, parseErrs)
	}
	if f == nil {
		t.Fatalf("parsed file is nil for block %d", blockNum)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("structural validation errors in block %d: %v", blockNum, structErrs)
	}

	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("semantic validation errors in block %d: %v", blockNum, semErrs)
	}
}

func validateFragmentBlock(t *testing.T, content, mdFile string, blockNum int) {
	t.Helper()

	// Wrap fragment with a synthetic package header so the parser accepts it.
	wrapped := fmt.Sprintf("package \"doc-fragment\" version \"0.0.0\" lang \"2.0\"\n\n%s", content)

	f, parseErrs := parser.Parse(wrapped, fmt.Sprintf("%s:fragment-%d", filepath.Base(mdFile), blockNum))
	if parseErrs != nil {
		t.Fatalf("parse errors in fragment block %d: %v", blockNum, parseErrs)
	}
	if f == nil {
		t.Fatalf("parsed file is nil for fragment block %d", blockNum)
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("structural validation errors in fragment block %d: %v", blockNum, structErrs)
	}

	// Skip semantic validation for fragments — they may reference resources
	// not defined in the synthetic wrapper.
}

func validateInvalidBlock(t *testing.T, content, mdFile string, blockNum int) {
	t.Helper()

	// An invalid block must produce at least one parse or validation error.
	f, parseErrs := parser.Parse(content, fmt.Sprintf("%s:invalid-%d", filepath.Base(mdFile), blockNum))
	if parseErrs != nil {
		// Expected — parse errors found.
		return
	}
	if f == nil {
		// Expected — nil AST.
		return
	}

	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		// Expected — structural errors found.
		return
	}

	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		// Expected — semantic errors found.
		return
	}

	t.Fatalf("invalid block %d in %s was expected to produce errors but validated successfully",
		blockNum, filepath.Base(mdFile))
}

func getRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// integration_tests/ is one level below the repo root
	return filepath.Dir(dir)
}
