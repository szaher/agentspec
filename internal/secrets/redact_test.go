package secrets

import (
	"bytes"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestRedactFilter_AddSecretThenRedactString(t *testing.T) {
	f := NewRedactFilter(slog.NewTextHandler(&bytes.Buffer{}, nil))
	f.AddSecret("super-secret-token")

	got := f.RedactString("the token is super-secret-token here")
	want := "the token is ***REDACTED*** here"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedactFilter_MultipleSecretsRedacted(t *testing.T) {
	f := NewRedactFilter(slog.NewTextHandler(&bytes.Buffer{}, nil))
	f.AddSecret("secret-a")
	f.AddSecret("secret-b")

	got := f.RedactString("values: secret-a and secret-b end")
	if strings.Contains(got, "secret-a") {
		t.Errorf("secret-a was not redacted: %q", got)
	}
	if strings.Contains(got, "secret-b") {
		t.Errorf("secret-b was not redacted: %q", got)
	}
	if !strings.Contains(got, "***REDACTED***") {
		t.Errorf("expected ***REDACTED*** placeholder in output: %q", got)
	}
}

func TestRedactFilter_NoSecretsReturnsOriginal(t *testing.T) {
	f := NewRedactFilter(slog.NewTextHandler(&bytes.Buffer{}, nil))

	input := "nothing to redact here"
	got := f.RedactString(input)
	if got != input {
		t.Errorf("got %q, want %q", got, input)
	}
}

func TestRedactFilter_AddEmptySecretIgnored(t *testing.T) {
	f := NewRedactFilter(slog.NewTextHandler(&bytes.Buffer{}, nil))
	f.AddSecret("")

	input := "still here"
	got := f.RedactString(input)
	if got != input {
		t.Errorf("got %q, want %q (empty secret should not affect output)", got, input)
	}
}

func TestRedactFilter_Handle_MessageRedacted(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		// Remove timestamp for deterministic output
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})

	f := NewRedactFilter(inner)
	f.AddSecret("my-api-key")

	logger := slog.New(f)
	logger.Info("connecting with my-api-key")

	output := buf.String()
	if strings.Contains(output, "my-api-key") {
		t.Errorf("secret was not redacted from message: %s", output)
	}
	if !strings.Contains(output, "***REDACTED***") {
		t.Errorf("expected ***REDACTED*** in output: %s", output)
	}
}

func TestRedactFilter_Handle_AttrValueRedacted(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	})

	f := NewRedactFilter(inner)
	f.AddSecret("secret-value-42")

	logger := slog.New(f)
	logger.Info("request completed", "token", "secret-value-42")

	output := buf.String()
	if strings.Contains(output, "secret-value-42") {
		t.Errorf("secret was not redacted from attr value: %s", output)
	}
	if !strings.Contains(output, "***REDACTED***") {
		t.Errorf("expected ***REDACTED*** in attr value: %s", output)
	}
}

func TestRedactFilter_WithAttrs_SharesSecrets(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, nil)

	f := NewRedactFilter(inner)
	f.AddSecret("shared-secret")

	child := f.WithAttrs([]slog.Attr{slog.String("component", "auth")})
	childFilter, ok := child.(*RedactFilter)
	if !ok {
		t.Fatalf("WithAttrs did not return *RedactFilter, got %T", child)
	}

	// The child should share the same secrets map
	got := childFilter.RedactString("contains shared-secret here")
	if strings.Contains(got, "shared-secret") {
		t.Errorf("child handler did not redact shared secret: %q", got)
	}

	// Adding a secret to the parent should be visible to the child
	f.AddSecret("another-secret")
	got2 := childFilter.RedactString("another-secret visible?")
	if strings.Contains(got2, "another-secret") {
		t.Errorf("secret added to parent not visible in child: %q", got2)
	}
}

func TestRedactFilter_WithGroup_SharesSecrets(t *testing.T) {
	var buf bytes.Buffer
	inner := slog.NewTextHandler(&buf, nil)

	f := NewRedactFilter(inner)
	f.AddSecret("group-secret")

	child := f.WithGroup("mygroup")
	childFilter, ok := child.(*RedactFilter)
	if !ok {
		t.Fatalf("WithGroup did not return *RedactFilter, got %T", child)
	}

	got := childFilter.RedactString("has group-secret inside")
	if strings.Contains(got, "group-secret") {
		t.Errorf("child handler did not redact group secret: %q", got)
	}

	// Adding a secret to the parent should be visible to the child
	f.AddSecret("new-group-secret")
	got2 := childFilter.RedactString("new-group-secret here")
	if strings.Contains(got2, "new-group-secret") {
		t.Errorf("secret added to parent not visible in group child: %q", got2)
	}
}

func TestRedactFilter_ThreadSafety(t *testing.T) {
	f := NewRedactFilter(slog.NewTextHandler(&bytes.Buffer{}, nil))

	var wg sync.WaitGroup
	const goroutines = 50

	// Half the goroutines add secrets, half redact strings
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				f.AddSecret("secret-" + string(rune('A'+n%26)))
			} else {
				_ = f.RedactString("test string with secret-A content")
			}
		}(i)
	}

	wg.Wait()
	// If we reach here without a data race, the test passes.
	// Run with -race to verify: go test -race ./internal/secrets/
}
