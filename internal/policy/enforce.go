package policy

import (
	"fmt"

	"github.com/szaher/designs/agentz/internal/ir"
	"github.com/szaher/designs/agentz/internal/validate"
)

// Enforce evaluates policies against IR resources and returns
// validation errors for any violations.
func Enforce(engine Engine, policies []ir.Policy, resources []ir.Resource) []*validate.ValidationError {
	violations := engine.Evaluate(policies, resources)
	var errs []*validate.ValidationError
	for _, v := range violations {
		errs = append(errs, &validate.ValidationError{
			File:    "",
			Line:    0,
			Column:  0,
			Message: fmt.Sprintf("policy violation on %s: %s", v.Resource.FQN, v.Message),
			Hint:    fmt.Sprintf("rule: %s %s %s", v.Rule.Action, v.Rule.Resource, v.Rule.Subject),
		})
	}
	return errs
}
