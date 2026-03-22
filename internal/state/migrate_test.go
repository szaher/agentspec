package state

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Successful roundtrip migration (local -> local)
// ---------------------------------------------------------------------------

func TestMigrate_Roundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.state.json")
	dstPath := filepath.Join(tmpDir, "dst.state.json")

	src := NewLocalBackend(srcPath)
	dst := NewLocalBackend(dstPath)

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{FQN: "agent.alpha", Hash: "h1", Status: StatusApplied, LastApplied: now, Adapter: "openai"},
		{FQN: "agent.beta", Hash: "h2", Status: StatusFailed, LastApplied: now, Adapter: "anthropic", Error: "timeout"},
		{FQN: "tool.gamma", Hash: "h3", Status: StatusApplied, LastApplied: now, Adapter: "openai"},
	}
	if err := src.Save(entries); err != nil {
		t.Fatalf("src.Save: %v", err)
	}

	result, err := Migrate(src, dst, false)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if result.Migrated != 3 {
		t.Errorf("Migrated = %d, want 3", result.Migrated)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
	if result.Duration <= 0 {
		t.Errorf("Duration = %v, want > 0", result.Duration)
	}
	if result.Source == "" {
		t.Error("Source is empty, want non-empty type string")
	}
	if result.Dest == "" {
		t.Error("Dest is empty, want non-empty type string")
	}

	// Verify entries were actually written to destination.
	loaded, err := dst.Load()
	if err != nil {
		t.Fatalf("dst.Load: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("dst has %d entries, want 3", len(loaded))
	}

	// Entries are sorted by FQN after Save, so verify order.
	wantFQNs := []string{"agent.alpha", "agent.beta", "tool.gamma"}
	for i, want := range wantFQNs {
		if loaded[i].FQN != want {
			t.Errorf("loaded[%d].FQN = %q, want %q", i, loaded[i].FQN, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Dry-run mode: entries counted but not written
// ---------------------------------------------------------------------------

func TestMigrate_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.state.json")
	dstPath := filepath.Join(tmpDir, "dst.state.json")

	src := NewLocalBackend(srcPath)
	dst := NewLocalBackend(dstPath)

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{FQN: "agent.one", Hash: "h1", Status: StatusApplied, LastApplied: now, Adapter: "a1"},
		{FQN: "agent.two", Hash: "h2", Status: StatusApplied, LastApplied: now, Adapter: "a2"},
	}
	if err := src.Save(entries); err != nil {
		t.Fatalf("src.Save: %v", err)
	}

	result, err := Migrate(src, dst, true)
	if err != nil {
		t.Fatalf("Migrate dry-run: %v", err)
	}

	if result.Migrated != 2 {
		t.Errorf("Migrated = %d, want 2", result.Migrated)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}

	// Destination should be empty (nothing written in dry-run).
	loaded, err := dst.Load()
	if err != nil {
		t.Fatalf("dst.Load: %v", err)
	}
	if loaded != nil {
		t.Errorf("dst has %d entries after dry-run, want nil", len(loaded))
	}
}

// ---------------------------------------------------------------------------
// Source load failure
// ---------------------------------------------------------------------------

func TestMigrate_SourceLoadFailure(t *testing.T) {
	tmpDir := t.TempDir()
	dstPath := filepath.Join(tmpDir, "dst.state.json")

	// Point source at a non-existent directory to cause a load error.
	src := &failingBackend{loadErr: errors.New("connection refused")}
	dst := NewLocalBackend(dstPath)

	result, err := Migrate(src, dst, false)
	if err == nil {
		t.Fatal("Migrate should return error when source load fails")
	}
	if !strings.Contains(err.Error(), "source load failed") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "source load failed")
	}
	if !strings.Contains(err.Error(), "connection refused") {
		t.Errorf("error = %q, want it to contain the wrapped cause", err.Error())
	}
	// Result should still be returned with metadata populated.
	if result == nil {
		t.Fatal("result should be non-nil even on source failure")
	}
	if result.Migrated != 0 {
		t.Errorf("Migrated = %d, want 0", result.Migrated)
	}
}

// ---------------------------------------------------------------------------
// Destination save failure
// ---------------------------------------------------------------------------

func TestMigrate_DestSaveFailure(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.state.json")

	src := NewLocalBackend(srcPath)

	now := time.Now().Truncate(time.Millisecond)
	entries := []Entry{
		{FQN: "agent.x", Hash: "hx", Status: StatusApplied, LastApplied: now, Adapter: "ax"},
		{FQN: "agent.y", Hash: "hy", Status: StatusApplied, LastApplied: now, Adapter: "ay"},
	}
	if err := src.Save(entries); err != nil {
		t.Fatalf("src.Save: %v", err)
	}

	// Use a backend that always fails on Save.
	dst := &failingBackend{saveErr: errors.New("disk full")}

	result, err := Migrate(src, dst, false)
	if err == nil {
		t.Fatal("Migrate should return error when destination save fails")
	}
	if result == nil {
		t.Fatal("result should be non-nil even on save failure")
	}
	if result.Failed != 2 {
		t.Errorf("Failed = %d, want 2", result.Failed)
	}
	if result.Migrated != 0 {
		t.Errorf("Migrated = %d, want 0", result.Migrated)
	}
	if len(result.Errors) != 2 {
		t.Errorf("len(Errors) = %d, want 2", len(result.Errors))
	}
	for _, e := range result.Errors {
		if !strings.Contains(e, "disk full") {
			t.Errorf("error entry = %q, want it to contain %q", e, "disk full")
		}
	}
}

// ---------------------------------------------------------------------------
// Empty source (zero entries)
// ---------------------------------------------------------------------------

func TestMigrate_EmptySource(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.state.json")
	dstPath := filepath.Join(tmpDir, "dst.state.json")

	src := NewLocalBackend(srcPath)
	dst := NewLocalBackend(dstPath)

	// Source has no state file, so Load returns nil entries.
	result, err := Migrate(src, dst, false)
	if err != nil {
		t.Fatalf("Migrate empty source: %v", err)
	}
	if result.Migrated != 0 {
		t.Errorf("Migrated = %d, want 0", result.Migrated)
	}
	if result.Failed != 0 {
		t.Errorf("Failed = %d, want 0", result.Failed)
	}
}

// ---------------------------------------------------------------------------
// Source and Dest fields are populated
// ---------------------------------------------------------------------------

func TestMigrate_ResultMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.state.json")
	dstPath := filepath.Join(tmpDir, "dst.state.json")

	src := NewLocalBackend(srcPath)
	dst := NewLocalBackend(dstPath)

	result, err := Migrate(src, dst, false)
	if err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	if !strings.Contains(result.Source, "LocalBackend") {
		t.Errorf("Source = %q, want it to contain %q", result.Source, "LocalBackend")
	}
	if !strings.Contains(result.Dest, "LocalBackend") {
		t.Errorf("Dest = %q, want it to contain %q", result.Dest, "LocalBackend")
	}
}

// ---------------------------------------------------------------------------
// Dry-run with empty source
// ---------------------------------------------------------------------------

func TestMigrate_DryRunEmptySource(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.state.json")
	dstPath := filepath.Join(tmpDir, "dst.state.json")

	src := NewLocalBackend(srcPath)
	dst := NewLocalBackend(dstPath)

	result, err := Migrate(src, dst, true)
	if err != nil {
		t.Fatalf("Migrate dry-run empty: %v", err)
	}
	if result.Migrated != 0 {
		t.Errorf("Migrated = %d, want 0", result.Migrated)
	}
}

// ---------------------------------------------------------------------------
// failingBackend: test helper that simulates backend failures
// ---------------------------------------------------------------------------

type failingBackend struct {
	loadErr error
	saveErr error
}

func (b *failingBackend) Load() ([]Entry, error) {
	if b.loadErr != nil {
		return nil, b.loadErr
	}
	return nil, nil
}

func (b *failingBackend) Save(_ []Entry) error {
	return b.saveErr
}

func (b *failingBackend) Get(_ string) (*Entry, error) {
	return nil, nil
}

func (b *failingBackend) List(_ *Status) ([]Entry, error) {
	return nil, nil
}
