package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

// Global state variable, to be managed by the application logic (e.g., loaded at startup).
// It's a pointer to allow modification by reference.
var (
	CurrentState *ArchiveProgressState
	mutex        sync.Mutex
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

// UpdateLastProcessedTopic is a placeholder to update the global CurrentState.
// The actual update logic will depend on how run_archiver.go manages CurrentState.
func UpdateLastProcessedTopic(subForumID, topicID string, pageNum int, stateFilePath string) {
	mutex.Lock()
	defer mutex.Unlock()

	if CurrentState == nil {
		log.Printf("[WARNING] STATE_PLACEHOLDER: UpdateLastProcessedTopic called but CurrentState is nil. Cannot update.")
		return
	}

	// For placeholder, just log. In reality, you'd update CurrentState fields.
	log.Printf("[INFO] STATE_PLACEHOLDER: Would update CurrentState: LastSubForumID=%s, LastTopicID=%s, LastPageNum=%d", subForumID, topicID, pageNum)
	// Example of actual update (currently commented out for placeholder):
	// CurrentState.LastProcessedSubForumID = subForumID
	// CurrentState.LastProcessedTopicID = topicID
	// CurrentState.LastProcessedPageNumberInTopic = pageNum
	// TODO: Add logic for ProcessedTopicIDsInCurrentSubForum and CompletedSubForumIDs if needed here or in SaveProgress
}

// SaveProgress is a placeholder to save the global CurrentState.
func SaveProgress(stateFilePath string) {
	mutex.Lock()
	defer mutex.Unlock() // Ensure mutex is unlocked even if CurrentState is nil or SaveState errors

	log.Printf("[INFO] STATE_PLACEHOLDER: SaveProgress called for path %s.", stateFilePath)
	if CurrentState == nil {
		log.Printf("[WARNING] STATE_PLACEHOLDER: CurrentState is nil. Nothing to save.")
		return
	}

	err := SaveState(CurrentState, stateFilePath)
	if err != nil {
		log.Printf("[ERROR] STATE_PLACEHOLDER: SaveProgress failed to save state: %v", err)
	} else {
		log.Printf("[INFO] STATE_PLACEHOLDER: SaveProgress successfully saved state to %s.", stateFilePath)
	}
}

// GetLastProcessedTopic returns the last processed topic and page for a sub-forum.
// ... existing code ...

// IsTopicCompleted checks if a topic is marked as completed.
// ... existing code ...
