package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SaveState saves the current archival progress to the specified file.
// It uses a safe-writing technique: write to a temporary file first, then rename.
func SaveState(state *ArchiveProgressState, filePath string) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	tempFilePath := filePath + ".tmp"

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory for state file %s: %w", dir, err)
	}

	if err := os.WriteFile(tempFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state to temporary file %s: %w", tempFilePath, err)
	}

	if err := os.Rename(tempFilePath, filePath); err != nil {
		return fmt.Errorf("failed to rename temporary state file %s to %s: %w", tempFilePath, filePath, err)
	}

	return nil
}

// LoadState loads the archival progress state from the specified file.
// If the file does not exist, it returns nil state and nil error, indicating a fresh start.
// If the file exists but cannot be parsed, it returns an error.
func LoadState(filePath string) (*ArchiveProgressState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state file, start fresh
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", filePath, err)
	}

	var state ArchiveProgressState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state from JSON in file %s: %w", filePath, err)
	}

	return &state, nil
}
