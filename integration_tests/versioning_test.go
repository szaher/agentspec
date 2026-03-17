package integration_tests

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/szaher/agentspec/internal/state"
)

// TestVersionSaveAndGet verifies saving and retrieving version history.
func TestVersionSaveAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	backend := state.NewLocalBackend(statePath)

	agentName := "test-agent"

	// Save 3 versions
	for i := 1; i <= 3; i++ {
		entry := state.VersionEntry{
			Version:   i,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Summary:   "Version " + string(rune('0'+i)),
			Snapshot: map[string]string{
				"model":  "claude-v" + string(rune('0'+i)),
				"prompt": "System prompt v" + string(rune('0'+i)),
			},
		}
		if err := backend.SaveVersion(agentName, entry); err != nil {
			t.Fatalf("SaveVersion %d failed: %v", i, err)
		}
	}

	// Retrieve versions
	versions, err := backend.GetVersions(agentName)
	if err != nil {
		t.Fatalf("GetVersions failed: %v", err)
	}

	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	// Verify versions are in order
	for i, v := range versions {
		expectedVersion := i + 1
		if v.Version != expectedVersion {
			t.Errorf("version %d: expected version number %d, got %d", i, expectedVersion, v.Version)
		}
	}

	// Verify snapshot data
	if versions[0].Snapshot["model"] != "claude-v1" {
		t.Errorf("version 1: expected model 'claude-v1', got %q", versions[0].Snapshot["model"])
	}
}

// TestVersionRetentionLimit verifies only last 10 versions are retained.
func TestVersionRetentionLimit(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	backend := state.NewLocalBackend(statePath)

	agentName := "test-agent"

	// Save 12 versions
	for i := 1; i <= 12; i++ {
		entry := state.VersionEntry{
			Version:   i,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Summary:   "Version X",
			Snapshot:  map[string]string{"version": string(rune('0' + (i % 10)))},
		}
		if err := backend.SaveVersion(agentName, entry); err != nil {
			t.Fatalf("SaveVersion %d failed: %v", i, err)
		}
	}

	// Retrieve versions
	versions, err := backend.GetVersions(agentName)
	if err != nil {
		t.Fatalf("GetVersions failed: %v", err)
	}

	// Should only retain last 10
	if len(versions) != 10 {
		t.Errorf("expected 10 versions (retention limit), got %d", len(versions))
	}

	// Verify we have versions 3-12 (first 2 should be dropped)
	if len(versions) > 0 && versions[0].Version != 3 {
		t.Errorf("expected first retained version to be 3, got %d", versions[0].Version)
	}

	if len(versions) == 10 && versions[9].Version != 12 {
		t.Errorf("expected last version to be 12, got %d", versions[9].Version)
	}
}

// TestRollbackCreatesNewVersion simulates rollback by saving a previous snapshot.
func TestRollbackCreatesNewVersion(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	backend := state.NewLocalBackend(statePath)

	agentName := "test-agent"

	// Save version 1
	v1 := state.VersionEntry{
		Version:   1,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Summary:   "Initial version",
		Snapshot: map[string]string{
			"model":  "claude-v1",
			"prompt": "Initial prompt",
		},
	}
	if err := backend.SaveVersion(agentName, v1); err != nil {
		t.Fatalf("SaveVersion 1 failed: %v", err)
	}

	// Save version 2
	v2 := state.VersionEntry{
		Version:   2,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Summary:   "Updated version",
		Snapshot: map[string]string{
			"model":  "claude-v2",
			"prompt": "Updated prompt",
		},
	}
	if err := backend.SaveVersion(agentName, v2); err != nil {
		t.Fatalf("SaveVersion 2 failed: %v", err)
	}

	// Simulate rollback: save a new version (3) with v1's snapshot
	v3Rollback := state.VersionEntry{
		Version:   3,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Summary:   "Rollback to version 1",
		Snapshot:  v1.Snapshot, // Same snapshot as v1
	}
	if err := backend.SaveVersion(agentName, v3Rollback); err != nil {
		t.Fatalf("SaveVersion 3 (rollback) failed: %v", err)
	}

	// Verify we now have 3 versions
	versions, err := backend.GetVersions(agentName)
	if err != nil {
		t.Fatalf("GetVersions failed: %v", err)
	}

	if len(versions) != 3 {
		t.Fatalf("expected 3 versions, got %d", len(versions))
	}

	// Verify v3 has the same snapshot as v1
	if versions[2].Version != 3 {
		t.Errorf("expected version 3, got %d", versions[2].Version)
	}

	if versions[2].Snapshot["model"] != "claude-v1" {
		t.Errorf("expected rollback to have model 'claude-v1', got %q", versions[2].Snapshot["model"])
	}

	if versions[2].Snapshot["prompt"] != "Initial prompt" {
		t.Errorf("expected rollback to have prompt 'Initial prompt', got %q", versions[2].Snapshot["prompt"])
	}
}

// TestVersionsForNonExistentAgent verifies empty result for unknown agent.
func TestVersionsForNonExistentAgent(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	backend := state.NewLocalBackend(statePath)

	versions, err := backend.GetVersions("unknown-agent")
	if err != nil {
		t.Fatalf("GetVersions failed: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("expected 0 versions for unknown agent, got %d", len(versions))
	}
}

// TestVersionTimestamps verifies timestamps are saved correctly.
func TestVersionTimestamps(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	backend := state.NewLocalBackend(statePath)

	agentName := "test-agent"

	now := time.Now().UTC()
	entry := state.VersionEntry{
		Version:   1,
		Timestamp: now.Format(time.RFC3339),
		Summary:   "Test version",
		Snapshot:  map[string]string{"key": "value"},
	}

	if err := backend.SaveVersion(agentName, entry); err != nil {
		t.Fatalf("SaveVersion failed: %v", err)
	}

	versions, err := backend.GetVersions(agentName)
	if err != nil {
		t.Fatalf("GetVersions failed: %v", err)
	}

	if len(versions) != 1 {
		t.Fatalf("expected 1 version, got %d", len(versions))
	}

	// Parse timestamp and verify it's close to now
	savedTime, err := time.Parse(time.RFC3339, versions[0].Timestamp)
	if err != nil {
		t.Fatalf("failed to parse timestamp: %v", err)
	}

	diff := now.Sub(savedTime)
	if diff < 0 {
		diff = -diff
	}

	// Should be within 1 second
	if diff > time.Second {
		t.Errorf("timestamp difference too large: %v", diff)
	}
}

// TestMultipleAgentsVersions verifies version isolation between agents.
func TestMultipleAgentsVersions(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, ".agentspec.state.json")

	backend := state.NewLocalBackend(statePath)

	// Save versions for agent1
	for i := 1; i <= 2; i++ {
		entry := state.VersionEntry{
			Version:   i,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Summary:   "Agent1 version",
			Snapshot:  map[string]string{"agent": "agent1"},
		}
		if err := backend.SaveVersion("agent1", entry); err != nil {
			t.Fatalf("SaveVersion agent1 failed: %v", err)
		}
	}

	// Save versions for agent2
	for i := 1; i <= 3; i++ {
		entry := state.VersionEntry{
			Version:   i,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Summary:   "Agent2 version",
			Snapshot:  map[string]string{"agent": "agent2"},
		}
		if err := backend.SaveVersion("agent2", entry); err != nil {
			t.Fatalf("SaveVersion agent2 failed: %v", err)
		}
	}

	// Verify agent1 has 2 versions
	versions1, err := backend.GetVersions("agent1")
	if err != nil {
		t.Fatalf("GetVersions agent1 failed: %v", err)
	}
	if len(versions1) != 2 {
		t.Errorf("expected 2 versions for agent1, got %d", len(versions1))
	}

	// Verify agent2 has 3 versions
	versions2, err := backend.GetVersions("agent2")
	if err != nil {
		t.Fatalf("GetVersions agent2 failed: %v", err)
	}
	if len(versions2) != 3 {
		t.Errorf("expected 3 versions for agent2, got %d", len(versions2))
	}

	// Verify snapshots are isolated
	if versions1[0].Snapshot["agent"] != "agent1" {
		t.Errorf("agent1 snapshot corrupted")
	}
	if versions2[0].Snapshot["agent"] != "agent2" {
		t.Errorf("agent2 snapshot corrupted")
	}
}
