package compiler

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/szaher/designs/agentz/internal/ast"
	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/plugins"
	runtimePkg "github.com/szaher/designs/agentz/internal/runtime"
)

// ---------- gapanalysis: DetectFeatures ----------

func TestDetectFeatures_AgentSkillPipeline(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{
				Kind: "Agent",
				Name: "my-agent",
				FQN:  "pkg/my-agent",
				Attributes: map[string]interface{}{
					"strategy": "react",
					"stream":   true,
				},
			},
			{
				Kind: "Skill",
				Name: "my-skill",
				FQN:  "pkg/my-skill",
				Attributes: map[string]interface{}{
					"tool": map[string]interface{}{
						"type": "inline",
					},
				},
			},
			{
				Kind: "Pipeline",
				Name: "my-pipeline",
				FQN:  "pkg/my-pipeline",
				Attributes: map[string]interface{}{
					"steps": []interface{}{
						map[string]interface{}{"parallel": true},
					},
				},
			},
			{
				Kind:       "Prompt",
				Name:       "my-prompt",
				FQN:        "pkg/my-prompt",
				Attributes: map[string]interface{}{},
			},
			{
				Kind:       "Type",
				Name:       "my-type",
				FQN:        "pkg/my-type",
				Attributes: map[string]interface{}{},
			},
		},
	}

	features := DetectFeatures(doc)
	featureNames := make(map[string]bool)
	for _, f := range features {
		featureNames[f.Name] = true
	}

	expected := []string{
		"agent", "loop_react", "streaming",
		"skill", "inline_tools",
		"pipeline_sequential", "pipeline_hierarchical",
		"prompt",
		"type_definitions",
	}
	for _, name := range expected {
		if !featureNames[name] {
			t.Errorf("expected feature %q not found in detected features", name)
		}
	}
}

func TestDetectFeatures_AgentWithAllAttributes(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{
				Kind: "Agent",
				Name: "full-agent",
				FQN:  "pkg/full-agent",
				Attributes: map[string]interface{}{
					"strategy":         "reflexion",
					"config_params":    []interface{}{"param1"},
					"validation_rules": []interface{}{"rule1"},
					"eval_cases":       []interface{}{"case1"},
					"delegates":        []interface{}{"other-agent"},
					"memory":           map[string]interface{}{"strategy": "buffer"},
					"fallback":         "fallback-agent",
				},
			},
		},
	}

	features := DetectFeatures(doc)
	featureNames := make(map[string]bool)
	for _, f := range features {
		featureNames[f.Name] = true
	}

	expected := []string{
		"agent", "loop_reflexion", "config_params", "validation_rules",
		"eval_cases", "delegation", "sessions", "error_handling",
	}
	for _, name := range expected {
		if !featureNames[name] {
			t.Errorf("expected feature %q not found", name)
		}
	}
}

func TestDetectFeatures_EmptyDocument(t *testing.T) {
	doc := &ir.Document{}
	features := DetectFeatures(doc)
	if len(features) != 0 {
		t.Errorf("expected 0 features for empty doc, got %d", len(features))
	}
}

// ---------- gapanalysis: AnalyzeGaps ----------

func TestAnalyzeGaps_AllLevels(t *testing.T) {
	features := []DetectedFeature{
		{Name: "agent", ResourceFQN: "pkg/a"},
		{Name: "loop_react", ResourceFQN: "pkg/a"},
		{Name: "streaming", ResourceFQN: "pkg/a"},
		{Name: "inline_tools", ResourceFQN: "pkg/a"},
	}
	featureMap := plugins.FeatureMap{
		"agent":      plugins.FeatureFull,
		"loop_react": plugins.FeaturePartial,
		"streaming":  plugins.FeatureEmulated,
		// inline_tools not in map => FeatureNone
	}

	warnings := AnalyzeGaps(features, featureMap)

	// "agent" is full => no warning
	// "loop_react" is partial => 1 warning
	// "streaming" is emulated => 1 warning
	// "inline_tools" is none => 1 warning
	if len(warnings) != 3 {
		t.Fatalf("expected 3 warnings, got %d", len(warnings))
	}

	found := map[plugins.FeatureSupportLevel]bool{}
	for _, w := range warnings {
		found[w.Level] = true
	}
	if !found[plugins.FeaturePartial] {
		t.Error("expected a partial warning")
	}
	if !found[plugins.FeatureEmulated] {
		t.Error("expected an emulated warning")
	}
	if !found[plugins.FeatureNone] {
		t.Error("expected a none warning")
	}
}

func TestAnalyzeGaps_AllSupported(t *testing.T) {
	features := []DetectedFeature{
		{Name: "agent", ResourceFQN: "pkg/a"},
	}
	featureMap := plugins.FeatureMap{
		"agent": plugins.FeatureFull,
	}
	warnings := AnalyzeGaps(features, featureMap)
	if len(warnings) != 0 {
		t.Errorf("expected 0 warnings when all features fully supported, got %d", len(warnings))
	}
}

// ---------- gapanalysis: GapWarningsToStrings ----------

func TestGapWarningsToStrings(t *testing.T) {
	warnings := []GapWarning{
		{
			Feature:    "streaming",
			Level:      plugins.FeaturePartial,
			Message:    "Feature \"streaming\" has partial support",
			Suggestion: "Check docs",
		},
		{
			Feature: "mcp_tools",
			Level:   plugins.FeatureNone,
			Message: "Not supported",
		},
	}

	strs := GapWarningsToStrings(warnings)
	if len(strs) != 2 {
		t.Fatalf("expected 2 strings, got %d", len(strs))
	}
	if !strings.Contains(strs[0], "[partial]") {
		t.Errorf("first warning should contain '[partial]', got %q", strs[0])
	}
	if !strings.Contains(strs[0], "Check docs") {
		t.Errorf("first warning should contain suggestion, got %q", strs[0])
	}
	if !strings.Contains(strs[1], "[none]") {
		t.Errorf("second warning should contain '[none]', got %q", strs[1])
	}
}

// ---------- safezone ----------

func TestParseSafeZones_GeneratedAndUser(t *testing.T) {
	content := `// --- AGENTSPEC GENERATED START ---
// Do not edit between these markers; changes will be overwritten on recompile
func main() {}
// --- AGENTSPEC GENERATED END ---
// --- USER CODE START ---
// Your custom code here is preserved across recompilations
func myCustom() {}
// --- USER CODE END ---
`
	zones := ParseSafeZones(content, "//")
	if len(zones) == 0 {
		t.Fatal("expected at least 1 zone")
	}

	var genCount, userCount int
	for _, z := range zones {
		switch z.Type {
		case "generated":
			genCount++
		case "user":
			userCount++
		}
	}
	if genCount != 1 {
		t.Errorf("expected 1 generated zone, got %d", genCount)
	}
	if userCount != 1 {
		t.Errorf("expected 1 user zone, got %d", userCount)
	}
}

func TestExtractUserCode(t *testing.T) {
	content := `// --- AGENTSPEC GENERATED START ---
// Do not edit between these markers; changes will be overwritten on recompile
generated code here
// --- AGENTSPEC GENERATED END ---
// --- USER CODE START ---
// Your custom code here is preserved across recompilations
my custom implementation
// --- USER CODE END ---
`
	userCode := ExtractUserCode(content, "//")
	if len(userCode) != 1 {
		t.Fatalf("expected 1 user code section, got %d", len(userCode))
	}
	if !strings.Contains(userCode[0], "my custom implementation") {
		t.Errorf("user code should contain custom implementation, got %q", userCode[0])
	}
}

func TestMergeWithUserCode(t *testing.T) {
	newContent := `// --- AGENTSPEC GENERATED START ---
// Do not edit between these markers; changes will be overwritten on recompile
new generated code
// --- AGENTSPEC GENERATED END ---
// --- USER CODE START ---
// Your custom code here is preserved across recompilations
placeholder
// --- USER CODE END ---
`
	existingUserCode := map[int]string{
		0: "// --- USER CODE START ---\n// Your custom code here is preserved across recompilations\nmy preserved code\n// --- USER CODE END ---\n",
	}

	merged := MergeWithUserCode(newContent, "//", existingUserCode)
	if !strings.Contains(merged, "my preserved code") {
		t.Error("merged output should contain preserved user code")
	}
	if !strings.Contains(merged, "new generated code") {
		t.Error("merged output should contain new generated code")
	}
}

func TestMergeWithUserCode_EmptyExisting(t *testing.T) {
	original := "some content"
	merged := MergeWithUserCode(original, "//", nil)
	if merged != original {
		t.Errorf("MergeWithUserCode with nil map should return original, got %q", merged)
	}
}

func TestWrapGenerated(t *testing.T) {
	result := WrapGenerated("func main() {}\n", "//")
	if !strings.Contains(result, "AGENTSPEC GENERATED START") {
		t.Error("WrapGenerated should contain start marker")
	}
	if !strings.Contains(result, "AGENTSPEC GENERATED END") {
		t.Error("WrapGenerated should contain end marker")
	}
	if !strings.Contains(result, "func main() {}") {
		t.Error("WrapGenerated should contain the original content")
	}
	if !strings.Contains(result, "Do not edit") {
		t.Error("WrapGenerated should contain edit warning")
	}
}

func TestWrapUserCode(t *testing.T) {
	result := WrapUserCode("// TODO: implement\n", "//")
	if !strings.Contains(result, "USER CODE START") {
		t.Error("WrapUserCode should contain start marker")
	}
	if !strings.Contains(result, "USER CODE END") {
		t.Error("WrapUserCode should contain end marker")
	}
	if !strings.Contains(result, "preserved across recompilations") {
		t.Error("WrapUserCode should contain preservation note")
	}
}

func TestCommentPrefixForLanguage(t *testing.T) {
	tests := []struct {
		lang string
		want string
	}{
		{"go", "//"},
		{"Go", "//"},
		{"python", "#"},
		{"py", "#"},
		{"javascript", "//"},
		{"js", "//"},
		{"typescript", "//"},
		{"ts", "//"},
		{"java", "//"},
		{"c", "//"},
		{"cpp", "//"},
		{"rust", "//"},
		{"yaml", "#"},
		{"yml", "#"},
		{"bash", "#"},
		{"sh", "#"},
		{"unknown", "#"}, // default
	}
	for _, tc := range tests {
		t.Run(tc.lang, func(t *testing.T) {
			got := CommentPrefixForLanguage(tc.lang)
			if got != tc.want {
				t.Errorf("CommentPrefixForLanguage(%q) = %q, want %q", tc.lang, got, tc.want)
			}
		})
	}
}

// ---------- cross ----------

func TestCurrentPlatform(t *testing.T) {
	plat := CurrentPlatform()
	if plat == "" {
		t.Fatal("CurrentPlatform should not return empty string")
	}
	if !strings.Contains(plat, "/") {
		t.Errorf("CurrentPlatform = %q, should contain '/'", plat)
	}
	expected := runtime.GOOS + "/" + runtime.GOARCH
	if plat != expected {
		t.Errorf("CurrentPlatform = %q, want %q", plat, expected)
	}
}

func TestValidatePlatform(t *testing.T) {
	tests := []struct {
		platform string
		wantErr  bool
	}{
		{"", false}, // empty is valid (uses current)
		{"linux/amd64", false},
		{"linux/arm64", false},
		{"darwin/amd64", false},
		{"darwin/arm64", false},
		{"windows/amd64", false},
		{"freebsd/amd64", true}, // unsupported
		{"linux/mips", true},    // unsupported
		{"invalid", true},       // invalid format
	}
	for _, tc := range tests {
		t.Run(tc.platform, func(t *testing.T) {
			err := ValidatePlatform(tc.platform)
			if tc.wantErr && err == nil {
				t.Errorf("expected error for platform %q, got nil", tc.platform)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error for platform %q: %v", tc.platform, err)
			}
		})
	}
}

// ---------- configref ----------

func TestGenerateConfigRef_WithParams(t *testing.T) {
	agents := []AgentConfigRef{
		{
			AgentName: "my-agent",
			Params: []runtimePkg.ConfigParamDef{
				{
					Name:        "api-key",
					Type:        "string",
					Description: "The API key",
					Required:    true,
					Secret:      true,
				},
				{
					Name:        "max-retries",
					Type:        "int",
					Description: "Max retry count",
					Required:    false,
					HasDefault:  true,
					Default:     "3",
				},
			},
		},
	}

	result := GenerateConfigRef(agents, "test-artifact")

	if !strings.Contains(result, "test-artifact") {
		t.Error("config ref should contain artifact name")
	}
	if !strings.Contains(result, "my-agent") {
		t.Error("config ref should contain agent name")
	}
	if !strings.Contains(result, "api-key") {
		t.Error("config ref should contain param name")
	}
	if !strings.Contains(result, "**Yes**") {
		t.Error("config ref should show required param")
	}
	if !strings.Contains(result, "AGENTSPEC_MY_AGENT_API_KEY") {
		t.Error("config ref should contain env variable name")
	}
	if !strings.Contains(result, "`3`") {
		t.Error("config ref should contain default value")
	}
	if !strings.Contains(result, "Quick Setup") {
		t.Error("config ref should contain setup section")
	}
	if !strings.Contains(result, "<secret>") {
		t.Error("config ref should show <secret> placeholder for secret params")
	}
}

func TestGenerateConfigRef_NoParams(t *testing.T) {
	agents := []AgentConfigRef{
		{AgentName: "simple-agent", Params: nil},
	}

	result := GenerateConfigRef(agents, "simple")

	if !strings.Contains(result, "simple-agent") {
		t.Error("config ref should contain agent name")
	}
	if !strings.Contains(result, "No configuration parameters declared") {
		t.Error("config ref should indicate no params")
	}
}

func TestGenerateConfigRef_EnvKeyFormat(t *testing.T) {
	agents := []AgentConfigRef{
		{
			AgentName: "my-complex-agent",
			Params: []runtimePkg.ConfigParamDef{
				{Name: "db-connection-string", Type: "string", Description: "DB conn"},
			},
		},
	}

	result := GenerateConfigRef(agents, "test")

	// Hyphens should become underscores, everything uppercase
	expectedEnvKey := "AGENTSPEC_MY_COMPLEX_AGENT_DB_CONNECTION_STRING"
	if !strings.Contains(result, expectedEnvKey) {
		t.Errorf("config ref should contain env key %q", expectedEnvKey)
	}
}

func TestDetectFeatures_ControlFlow(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{
				Kind: "Agent",
				Name: "flow-agent",
				FQN:  "pkg/flow-agent",
				Attributes: map[string]interface{}{
					"on_input": []interface{}{
						map[string]interface{}{"type": "if"},
						map[string]interface{}{"type": "for_each"},
					},
				},
			},
		},
	}

	features := DetectFeatures(doc)
	featureNames := make(map[string]bool)
	for _, f := range features {
		featureNames[f.Name] = true
	}

	if !featureNames["control_flow_if"] {
		t.Error("expected control_flow_if feature")
	}
	if !featureNames["control_flow_foreach"] {
		t.Error("expected control_flow_foreach feature")
	}
}

func TestDetectFeatures_MCPTools(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{
				Kind: "Skill",
				Name: "mcp-skill",
				FQN:  "pkg/mcp-skill",
				Attributes: map[string]interface{}{
					"tool": map[string]interface{}{
						"type": "mcp",
					},
				},
			},
		},
	}

	features := DetectFeatures(doc)
	featureNames := make(map[string]bool)
	for _, f := range features {
		featureNames[f.Name] = true
	}

	if !featureNames["mcp_tools"] {
		t.Error("expected mcp_tools feature")
	}
}

func TestDetectFeatures_PromptVariables(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{
				Kind: "Prompt",
				Name: "templ-prompt",
				FQN:  "pkg/templ-prompt",
				Attributes: map[string]interface{}{
					"variables": map[string]interface{}{"name": "string"},
				},
			},
		},
	}

	features := DetectFeatures(doc)
	featureNames := make(map[string]bool)
	for _, f := range features {
		featureNames[f.Name] = true
	}

	if !featureNames["prompt"] {
		t.Error("expected prompt feature")
	}
	if !featureNames["prompt_variables"] {
		t.Error("expected prompt_variables feature")
	}
}

func TestDetectFeatures_AllStrategies(t *testing.T) {
	strategies := []struct {
		strategy    string
		featureName string
	}{
		{"react", "loop_react"},
		{"plan-and-execute", "loop_plan_execute"},
		{"plan_and_execute", "loop_plan_execute"},
		{"reflexion", "loop_reflexion"},
		{"router", "loop_router"},
		{"map-reduce", "loop_map_reduce"},
		{"map_reduce", "loop_map_reduce"},
	}

	for _, tc := range strategies {
		t.Run(tc.strategy, func(t *testing.T) {
			doc := &ir.Document{
				Resources: []ir.Resource{
					{
						Kind: "Agent",
						Name: "a",
						FQN:  "pkg/a",
						Attributes: map[string]interface{}{
							"strategy": tc.strategy,
						},
					},
				},
			}
			features := DetectFeatures(doc)
			found := false
			for _, f := range features {
				if f.Name == tc.featureName {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("strategy %q should detect feature %q", tc.strategy, tc.featureName)
			}
		})
	}
}

// ---------- compiler.go: agentNames ----------

func TestAgentNames(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{Kind: "Agent", Name: "alice"},
			{Kind: "Skill", Name: "search"},
			{Kind: "Agent", Name: "bob"},
			{Kind: "Pipeline", Name: "flow"},
		},
	}
	names := agentNames(doc)
	if len(names) != 2 {
		t.Fatalf("expected 2 agent names, got %d", len(names))
	}
	if names[0] != "alice" || names[1] != "bob" {
		t.Errorf("got names %v, want [alice bob]", names)
	}
}

func TestAgentNames_Empty(t *testing.T) {
	doc := &ir.Document{}
	names := agentNames(doc)
	if len(names) != 0 {
		t.Errorf("expected 0 agent names, got %d", len(names))
	}
}

// ---------- compiler.go: hasImports ----------

func TestHasImports_WithStatementImport(t *testing.T) {
	f := &ast.File{
		Statements: []ast.Statement{
			&ast.Import{Path: "github.com/example/pkg"},
		},
	}
	if !hasImports(f) {
		t.Error("expected hasImports to return true for file with Import statement")
	}
}

func TestHasImports_WithPackageImports(t *testing.T) {
	f := &ast.File{
		Package: &ast.Package{
			Imports: []*ast.Import{
				{Path: "github.com/example/pkg"},
			},
		},
	}
	if !hasImports(f) {
		t.Error("expected hasImports to return true for file with package imports")
	}
}

func TestHasImports_NoImports(t *testing.T) {
	f := &ast.File{}
	if hasImports(f) {
		t.Error("expected hasImports to return false for file with no imports")
	}
}

// ---------- embed.go: generateMain ----------

func TestGenerateMain(t *testing.T) {
	data := TemplateData{
		Version:   "1.2.3",
		BuildTime: "2025-01-01T00:00:00Z",
		Target:    "standalone",
	}
	src := generateMain(data)
	if !strings.Contains(src, "1.2.3") {
		t.Error("generated main should contain version")
	}
	if !strings.Contains(src, "2025-01-01T00:00:00Z") {
		t.Error("generated main should contain build time")
	}
	if !strings.Contains(src, "standalone") {
		t.Error("generated main should contain target")
	}
	if !strings.Contains(src, "package main") {
		t.Error("generated main should start with package main")
	}
}

// ---------- embed.go: buildEnv ----------

func TestBuildEnv_WithPlatform(t *testing.T) {
	env := buildEnv("linux/arm64")

	goosFound := false
	goarchFound := false
	for _, e := range env {
		if e == "GOOS=linux" {
			goosFound = true
		}
		if e == "GOARCH=arm64" {
			goarchFound = true
		}
	}
	if !goosFound {
		t.Error("expected GOOS=linux in build env")
	}
	if !goarchFound {
		t.Error("expected GOARCH=arm64 in build env")
	}
}

func TestBuildEnv_EmptyPlatform(t *testing.T) {
	env := buildEnv("")
	// Should return os.Environ() as-is
	if len(env) != len(os.Environ()) {
		t.Errorf("buildEnv(\"\") returned %d env vars, expected %d", len(env), len(os.Environ()))
	}
}

func TestBuildEnv_InvalidFormat(t *testing.T) {
	env := buildEnv("invalid")
	// Should return os.Environ() as-is since no "/" in platform
	if len(env) != len(os.Environ()) {
		t.Errorf("buildEnv(\"invalid\") returned %d env vars, expected %d", len(env), len(os.Environ()))
	}
}

// ---------- gapanalysis: gapSuggestion ----------

func TestGapSuggestion_NoneLevel(t *testing.T) {
	tests := []struct {
		feature      string
		wantContains string
	}{
		{"loop_reflexion", "LangGraph"},
		{"loop_router", "LangGraph"},
		{"loop_map_reduce", "LangGraph"},
		{"pipeline_conditional", "LangGraph"},
		{"inline_tools", "command or HTTP"},
		{"mcp_tools", "HTTP or command"},
		{"control_flow_if", "standalone or LangGraph"},
		{"control_flow_foreach", "standalone or LangGraph"},
		{"unknown_feature", "omitted"},
	}
	for _, tc := range tests {
		t.Run(tc.feature, func(t *testing.T) {
			suggestion := gapSuggestion(tc.feature, plugins.FeatureNone)
			if !strings.Contains(suggestion, tc.wantContains) {
				t.Errorf("gapSuggestion(%q, none) = %q, want to contain %q", tc.feature, suggestion, tc.wantContains)
			}
		})
	}
}

func TestGapSuggestion_EmulatedLevel(t *testing.T) {
	s := gapSuggestion("agent", plugins.FeatureEmulated)
	if !strings.Contains(s, "application-level") {
		t.Errorf("expected emulated suggestion to mention application-level, got %q", s)
	}
}

func TestGapSuggestion_PartialLevel(t *testing.T) {
	s := gapSuggestion("agent", plugins.FeaturePartial)
	if !strings.Contains(s, "documentation") {
		t.Errorf("expected partial suggestion to mention documentation, got %q", s)
	}
}

// ---------- safezone: ParseSafeZones edge cases ----------

func TestParseSafeZones_OutsideContent(t *testing.T) {
	content := "line 1\nline 2\nline 3\n"
	zones := ParseSafeZones(content, "//")
	if len(zones) != 1 {
		t.Fatalf("expected 1 outside zone, got %d", len(zones))
	}
	if zones[0].Type != "outside" {
		t.Errorf("zone type = %q, want %q", zones[0].Type, "outside")
	}
}

func TestParseSafeZones_MultipleZones(t *testing.T) {
	content := `preamble
// --- AGENTSPEC GENERATED START ---
// Do not edit between these markers; changes will be overwritten on recompile
gen code
// --- AGENTSPEC GENERATED END ---
middle
// --- USER CODE START ---
// Your custom code here is preserved across recompilations
user code
// --- USER CODE END ---
footer
`
	zones := ParseSafeZones(content, "//")
	// Should have: outside (preamble), generated, outside (middle), user, outside (footer)
	if len(zones) < 4 {
		t.Fatalf("expected at least 4 zones, got %d", len(zones))
	}
	types := make([]string, len(zones))
	for i, z := range zones {
		types[i] = z.Type
	}
	// Verify the sequence includes generated and user zones
	hasGen := false
	hasUser := false
	for _, tp := range types {
		if tp == "generated" {
			hasGen = true
		}
		if tp == "user" {
			hasUser = true
		}
	}
	if !hasGen {
		t.Error("expected a generated zone")
	}
	if !hasUser {
		t.Error("expected a user zone")
	}
}

// ---------- configref: edge cases ----------

func TestConfigEnvKey(t *testing.T) {
	tests := []struct {
		agent string
		param string
		want  string
	}{
		{"my-agent", "api-key", "AGENTSPEC_MY_AGENT_API_KEY"},
		{"simple", "port", "AGENTSPEC_SIMPLE_PORT"},
		{"a-b-c", "x-y-z", "AGENTSPEC_A_B_C_X_Y_Z"},
	}
	for _, tc := range tests {
		t.Run(tc.agent+"_"+tc.param, func(t *testing.T) {
			got := configEnvKey(tc.agent, tc.param)
			if got != tc.want {
				t.Errorf("configEnvKey(%q, %q) = %q, want %q", tc.agent, tc.param, got, tc.want)
			}
		})
	}
}

// ---------- Pipeline conditional detection edge case ----------

func TestDetectFeatures_PipelineConditional(t *testing.T) {
	doc := &ir.Document{
		Resources: []ir.Resource{
			{
				Kind: "Pipeline",
				Name: "cond-pipeline",
				FQN:  "pkg/cond",
				Attributes: map[string]interface{}{
					"steps": []interface{}{
						map[string]interface{}{"when": "input.length > 100"},
					},
				},
			},
		},
	}
	features := DetectFeatures(doc)
	found := false
	for _, f := range features {
		if f.Name == "pipeline_conditional" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected pipeline_conditional feature for step with 'when' clause")
	}
}

// ---------- Compile validation ----------

func TestCompile_NoInputFiles(t *testing.T) {
	_, err := Compile(nil, CompileOptions{})
	if err == nil {
		t.Fatal("expected error for no input files")
	}
	if !strings.Contains(err.Error(), "no input files") {
		t.Errorf("error %q should contain 'no input files'", err.Error())
	}
}

func TestCompile_UnsupportedPlatform(t *testing.T) {
	_, err := Compile([]string{"test.ias"}, CompileOptions{
		Platform: "freebsd/mips",
	})
	if err == nil {
		t.Fatal("expected error for unsupported platform")
	}
	if !strings.Contains(err.Error(), "unsupported platform") {
		t.Errorf("error %q should contain 'unsupported platform'", err.Error())
	}
}

// ---------- moduleRoot ----------

func TestModuleRoot(t *testing.T) {
	root := moduleRoot()
	if root == "" {
		t.Fatal("moduleRoot should not return empty string")
	}
	// Should find a go.mod somewhere
	if root != "." {
		// It found a real module root
		if !strings.Contains(root, "/") {
			t.Logf("moduleRoot = %q (may be relative)", root)
		}
	}
}

// ---------- defaultSearchPaths ----------

func TestDefaultSearchPaths(t *testing.T) {
	paths := defaultSearchPaths()
	// Should return at least the agentspec and legacy agentz paths
	if len(paths) < 2 {
		t.Errorf("expected at least 2 search paths, got %d", len(paths))
	}
	foundAgentspec := false
	foundLegacy := false
	for _, p := range paths {
		if strings.Contains(p, ".agentspec") {
			foundAgentspec = true
		}
		if strings.Contains(p, ".agentz") {
			foundLegacy = true
		}
	}
	if !foundAgentspec {
		t.Error("expected .agentspec path in search paths")
	}
	if !foundLegacy {
		t.Error("expected .agentz legacy path in search paths")
	}
}
