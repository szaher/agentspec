package integration_tests

import (
	"testing"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/parser"
	"github.com/szaher/designs/agentz/internal/validate"
)

func TestMultiEnvironmentPlan(t *testing.T) {
	input := readTestFile(t, "testdata/multi_env.ias")

	f, parseErrs := parser.Parse(input, "multi_env.ias")
	if parseErrs != nil {
		t.Fatalf("parse errors: %v", parseErrs)
	}

	// Structural and semantic validation
	structErrs := validate.ValidateStructural(f)
	if len(structErrs) > 0 {
		t.Fatalf("structural errors: %v", structErrs)
	}
	semErrs := validate.ValidateSemantic(f)
	if len(semErrs) > 0 {
		t.Fatalf("semantic errors: %v", semErrs)
	}

	// Lower to IR
	doc, err := ir.Lower(f)
	if err != nil {
		t.Fatalf("IR lowering: %v", err)
	}

	// Apply dev environment
	devDoc, err := ir.ApplyEnvironment(doc, "dev")
	if err != nil {
		t.Fatalf("apply dev environment: %v", err)
	}

	// Find agent in dev doc
	var devAgent *ir.Resource
	for i := range devDoc.Resources {
		if devDoc.Resources[i].Kind == "Agent" && devDoc.Resources[i].Name == "assistant" {
			devAgent = &devDoc.Resources[i]
			break
		}
	}
	if devAgent == nil {
		t.Fatal("agent 'assistant' not found in dev environment")
	}
	if model, ok := devAgent.Attributes["model"].(string); !ok || model != "claude-haiku-latest" {
		t.Errorf("dev agent model: expected 'claude-haiku-latest', got %v", devAgent.Attributes["model"])
	}

	// Apply prod environment
	prodDoc, err := ir.ApplyEnvironment(doc, "prod")
	if err != nil {
		t.Fatalf("apply prod environment: %v", err)
	}

	var prodAgent *ir.Resource
	for i := range prodDoc.Resources {
		if prodDoc.Resources[i].Kind == "Agent" && prodDoc.Resources[i].Name == "assistant" {
			prodAgent = &prodDoc.Resources[i]
			break
		}
	}
	if prodAgent == nil {
		t.Fatal("agent 'assistant' not found in prod environment")
	}
	if model, ok := prodAgent.Attributes["model"].(string); !ok || model != "claude-sonnet-4-20250514" {
		t.Errorf("prod agent model: expected 'claude-sonnet-4-20250514', got %v", prodAgent.Attributes["model"])
	}

	// Dev and prod should produce different hashes for the agent
	if devAgent.Hash == prodAgent.Hash {
		t.Error("dev and prod agent hashes should differ when models differ")
	}
}
