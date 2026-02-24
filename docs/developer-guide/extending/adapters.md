# Custom Adapter Guide

This guide walks through implementing a custom adapter for the AgentSpec toolchain. By the end, you will understand the `Adapter` interface, the registration mechanism, and how to build, test, and integrate a new adapter.

## Adapter Interface

Every adapter must implement the `adapters.Adapter` interface defined in `internal/adapters/adapter.go`:

```go
type Adapter interface {
    Name() string
    Validate(ctx context.Context, resources []ir.Resource) error
    Apply(ctx context.Context, actions []Action) ([]Result, error)
    Export(ctx context.Context, resources []ir.Resource, outDir string) error
    Status(ctx context.Context) ([]ResourceStatus, error)
    Logs(ctx context.Context, w io.Writer, opts LogOptions) error
    Destroy(ctx context.Context) ([]Result, error)
}
```

See the [Adapter System architecture page](../architecture/adapters.md) for a full description of each method and the associated types.

## Step-by-Step: Building a "Lambda" Adapter

This example implements a hypothetical adapter that deploys agent skills as AWS Lambda functions.

### 1. Create the Package

Create a new directory under `internal/adapters/`:

```text
internal/adapters/lambda/
  lambda.go
  lambda_test.go
```

### 2. Define the Adapter Struct

```go
package lambda

import (
    "context"
    "fmt"
    "io"

    "github.com/szaher/designs/agentz/internal/adapters"
    "github.com/szaher/designs/agentz/internal/ir"
)

// Adapter deploys agent skills as AWS Lambda functions.
type Adapter struct {
    region    string
    role      string
    deployed  map[string]string // FQN -> function ARN
}

// New creates a new Lambda adapter.
func New(region, role string) *Adapter {
    return &Adapter{
        region:   region,
        role:     role,
        deployed: make(map[string]string),
    }
}
```

### 3. Implement the Interface

```go
// Name returns the adapter identifier.
func (a *Adapter) Name() string {
    return "lambda"
}

// Validate checks that all resources are compatible with Lambda deployment.
func (a *Adapter) Validate(ctx context.Context, resources []ir.Resource) error {
    for _, r := range resources {
        if r.Kind != "skill" && r.Kind != "agent" {
            continue
        }
        // Check that skills have a supported execution type
        if r.Kind == "skill" {
            toolType, _ := r.Attributes["tool_type"].(string)
            if toolType == "inline" {
                lang, _ := r.Attributes["language"].(string)
                if lang != "python" && lang != "javascript" {
                    return fmt.Errorf(
                        "lambda adapter: skill %q uses unsupported language %q (need python or javascript)",
                        r.Name, lang,
                    )
                }
            }
        }
    }
    return nil
}

// Apply executes the planned actions against AWS Lambda.
func (a *Adapter) Apply(ctx context.Context, actions []adapters.Action) ([]adapters.Result, error) {
    var results []adapters.Result

    for _, action := range actions {
        var result adapters.Result
        result.FQN = action.FQN
        result.Action = action.Type

        switch action.Type {
        case adapters.ActionCreate:
            arn, err := a.createFunction(ctx, action.Resource)
            if err != nil {
                result.Status = adapters.ResultFailed
                result.Error = err.Error()
            } else {
                result.Status = adapters.ResultSuccess
                result.Artifact = arn
                a.deployed[action.FQN] = arn
            }

        case adapters.ActionUpdate:
            err := a.updateFunction(ctx, action.Resource)
            if err != nil {
                result.Status = adapters.ResultFailed
                result.Error = err.Error()
            } else {
                result.Status = adapters.ResultSuccess
            }

        case adapters.ActionDelete:
            err := a.deleteFunction(ctx, action.FQN)
            if err != nil {
                result.Status = adapters.ResultFailed
                result.Error = err.Error()
            } else {
                result.Status = adapters.ResultSuccess
                delete(a.deployed, action.FQN)
            }

        case adapters.ActionNoop:
            result.Status = adapters.ResultSuccess
        }

        results = append(results, result)
    }

    return results, nil
}

// Export generates Lambda deployment artifacts (SAM template, function code).
func (a *Adapter) Export(ctx context.Context, resources []ir.Resource, outDir string) error {
    // Generate template.yaml and function code files
    // ...
    return nil
}

// Status returns the runtime status of deployed Lambda functions.
func (a *Adapter) Status(ctx context.Context) ([]adapters.ResourceStatus, error) {
    var statuses []adapters.ResourceStatus
    for fqn, arn := range a.deployed {
        statuses = append(statuses, adapters.ResourceStatus{
            FQN:   fqn,
            Name:  arn,
            Kind:  "lambda-function",
            State: "running",
        })
    }
    return statuses, nil
}

// Logs streams CloudWatch logs for deployed functions.
func (a *Adapter) Logs(ctx context.Context, w io.Writer, opts adapters.LogOptions) error {
    // Stream from CloudWatch Logs
    // ...
    return nil
}

// Destroy tears down all deployed Lambda functions.
func (a *Adapter) Destroy(ctx context.Context) ([]adapters.Result, error) {
    var results []adapters.Result
    for fqn := range a.deployed {
        err := a.deleteFunction(ctx, fqn)
        result := adapters.Result{
            FQN:    fqn,
            Action: adapters.ActionDelete,
        }
        if err != nil {
            result.Status = adapters.ResultFailed
            result.Error = err.Error()
        } else {
            result.Status = adapters.ResultSuccess
        }
        results = append(results, result)
    }
    a.deployed = make(map[string]string)
    return results, nil
}

// Internal helper methods
func (a *Adapter) createFunction(ctx context.Context, r *ir.Resource) (string, error) {
    // Call AWS SDK to create the Lambda function
    // Return the function ARN
    return fmt.Sprintf("arn:aws:lambda:%s:123456:function:%s", a.region, r.Name), nil
}

func (a *Adapter) updateFunction(ctx context.Context, r *ir.Resource) error {
    // Call AWS SDK to update the function code/config
    return nil
}

func (a *Adapter) deleteFunction(ctx context.Context, fqn string) error {
    // Call AWS SDK to delete the function
    return nil
}
```

### 4. Register the Adapter

Register the adapter factory so the plan engine can find it by name:

```go
func init() {
    adapters.Register("lambda", func() adapters.Adapter {
        // Read config from environment or defaults
        region := os.Getenv("AWS_REGION")
        if region == "" {
            region = "us-east-1"
        }
        role := os.Getenv("LAMBDA_ROLE_ARN")
        return New(region, role)
    })
}
```

### 5. Import in the CLI

Add a blank import in the CLI entrypoint to trigger the `init()` registration:

```go
// cmd/agentspec/main.go
import (
    _ "github.com/szaher/designs/agentz/internal/adapters/lambda"
)
```

## Testing

### Unit Tests

Test each method independently using mock AWS calls:

```go
func TestLambdaAdapter_Apply_Create(t *testing.T) {
    adapter := New("us-east-1", "arn:aws:iam::123456:role/test")

    actions := []adapters.Action{
        {
            FQN:  "test-app/skill/processor",
            Type: adapters.ActionCreate,
            Resource: &ir.Resource{
                Kind: "skill",
                Name: "processor",
                FQN:  "test-app/skill/processor",
                Attributes: map[string]interface{}{
                    "tool_type": "inline",
                    "language":  "python",
                },
            },
            Reason: "resource does not exist",
        },
    }

    results, err := adapter.Apply(context.Background(), actions)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if len(results) != 1 {
        t.Fatalf("expected 1 result, got %d", len(results))
    }

    if results[0].Status != adapters.ResultSuccess {
        t.Errorf("expected success, got %s: %s", results[0].Status, results[0].Error)
    }
}
```

### Validation Tests

Test that `Validate()` rejects incompatible resources:

```go
func TestLambdaAdapter_Validate_UnsupportedLanguage(t *testing.T) {
    adapter := New("us-east-1", "")

    resources := []ir.Resource{
        {
            Kind: "skill",
            Name: "rust-skill",
            Attributes: map[string]interface{}{
                "tool_type": "inline",
                "language":  "rust",
            },
        },
    }

    err := adapter.Validate(context.Background(), resources)
    if err == nil {
        t.Error("expected validation error for unsupported language")
    }
}
```

### Integration Tests

For real AWS integration tests, guard with an environment variable:

```go
func TestLambdaAdapter_Integration(t *testing.T) {
    if os.Getenv("AGENTSPEC_TEST_LAMBDA") == "" {
        t.Skip("set AGENTSPEC_TEST_LAMBDA=1 to run Lambda integration tests")
    }
    // Test against real AWS
}
```

## Using the Custom Adapter

Once registered, the adapter can be used in `.ias` files via a deploy target:

```text
deploy "production" target "lambda" {
    default true
}
```

Or via the legacy binding syntax:

```text
binding "aws" {
    adapter "lambda"
    default true
}
```

The plan engine will resolve the adapter by name and dispatch actions to it.
