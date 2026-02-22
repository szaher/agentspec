package state

import (
	"encoding/json"
	"errors"
	"os"
	"sort"
)

// LocalBackend implements Backend using a local JSON file.
type LocalBackend struct {
	Path string
}

// NewLocalBackend creates a new local JSON state backend.
func NewLocalBackend(path string) *LocalBackend {
	return &LocalBackend{Path: path}
}

// stateFile is the on-disk JSON structure.
type stateFile struct {
	Version string  `json:"version"`
	Entries []Entry `json:"entries"`
}

// Load reads all state entries from the JSON file.
func (b *LocalBackend) Load() ([]Entry, error) {
	data, err := os.ReadFile(b.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var sf stateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, err
	}
	return sf.Entries, nil
}

// Save writes all state entries to the JSON file with sorted keys.
func (b *LocalBackend) Save(entries []Entry) error {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].FQN < entries[j].FQN
	})
	sf := stateFile{
		Version: "1.0",
		Entries: entries,
	}
	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(b.Path, data, 0644)
}

// Get retrieves a single entry by FQN.
func (b *LocalBackend) Get(fqn string) (*Entry, error) {
	entries, err := b.Load()
	if err != nil {
		return nil, err
	}
	for i := range entries {
		if entries[i].FQN == fqn {
			return &entries[i], nil
		}
	}
	return nil, nil
}

// List returns all entries, optionally filtered by status.
func (b *LocalBackend) List(status *Status) ([]Entry, error) {
	entries, err := b.Load()
	if err != nil {
		return nil, err
	}
	if status == nil {
		return entries, nil
	}
	var filtered []Entry
	for _, e := range entries {
		if e.Status == *status {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}
