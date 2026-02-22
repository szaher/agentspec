package validate

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/ast"
)

// ValidateEnvironments performs environment-specific validation:
// conflicting overlay detection, valid resource references.
func ValidateEnvironments(f *ast.File) []*ValidationError {
	var errs []*ValidationError

	// Collect all resource names by kind/name
	resourceNames := make(map[string]bool)
	for _, stmt := range f.Statements {
		switch s := stmt.(type) {
		case *ast.Agent:
			resourceNames["agent/"+s.Name] = true
		case *ast.Prompt:
			resourceNames["prompt/"+s.Name] = true
		case *ast.Skill:
			resourceNames["skill/"+s.Name] = true
		case *ast.MCPServer:
			resourceNames["server/"+s.Name] = true
		case *ast.MCPClient:
			resourceNames["client/"+s.Name] = true
		case *ast.Secret:
			resourceNames["secret/"+s.Name] = true
		}
	}

	for _, stmt := range f.Statements {
		env, ok := stmt.(*ast.Environment)
		if !ok {
			continue
		}

		// Check for conflicting overrides (same resource+attribute)
		seen := make(map[string]bool)
		for _, ov := range env.Overrides {
			key := ov.Resource + "." + ov.Attribute
			if seen[key] {
				errs = append(errs, posError(ov.StartPos,
					fmt.Sprintf("conflicting override in environment %q: %s.%s set multiple times",
						env.Name, ov.Resource, ov.Attribute),
					"each attribute may only be overridden once per environment"))
			}
			seen[key] = true

			// Check that the referenced resource exists
			// Override resources are in format "kind/name"
			if !resourceNames[ov.Resource] {
				errs = append(errs, posError(ov.StartPos,
					fmt.Sprintf("environment %q: override target %q not found",
						env.Name, ov.Resource),
					"check the resource kind and name"))
			}
		}
	}

	return errs
}
