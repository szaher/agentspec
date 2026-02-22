package events

import (
	"encoding/json"
	"os"
)

// ExportLog writes collected events to a JSON file.
func ExportLog(events []*Event, path string) error {
	data, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
}
