package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFilePath := filepath.Join(tempDir, "test_archive_progress.json")
	tempStateFilePath := stateFilePath + ".tmp"

	initialState := &ArchiveProgressState{
		LastProcessedSubForumID:            "subforum1",
		LastProcessedTopicID:               "topic123",
		LastProcessedPageNumberInTopic:     5,
		ProcessedTopicIDsInCurrentSubForum: []string{"topic100", "topic101"},
		CompletedSubForumIDs:               []string{"subforum_A", "subforum_B"},
	}

	// Test SaveState
	if err := SaveState(initialState, stateFilePath); err != nil {
		t.Fatalf("SaveState() error = %v", err)
	}

	// Verify .tmp file is removed after successful save
	if _, err := os.Stat(tempStateFilePath); !os.IsNotExist(err) {
		t.Errorf("Temporary state file %s was not removed after saving", tempStateFilePath)
	}

	// Test LoadState - successful load
	loadedState, err := LoadState(stateFilePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if !reflect.DeepEqual(loadedState, initialState) {
		t.Errorf("LoadState() got = %v, want %v", loadedState, initialState)
	}

	// Test LoadState - file not found
	nonExistentFilePath := filepath.Join(tempDir, "does_not_exist.json")
	state, err := LoadState(nonExistentFilePath)
	if err != nil {
		t.Errorf("LoadState() for non-existent file error = %v, want nil", err)
	}
	if state != nil {
		t.Errorf("LoadState() for non-existent file got state = %v, want nil", state)
	}

	// Test SaveState - safe writing: simulate error during rename by making target read-only (tricky to do reliably cross-platform)
	// Instead, we check if the .tmp file is created and then original is updated.
	// For a more direct test of rename failure, one might need to mock os.Rename or use platform-specific file locking.

	// Test corruption during save (safe write check)
	// 1. Save initial state successfully (already done)
	// 2. Create a .tmp file manually as if a save started but didn't finish rename
	badTempStateFilePath := stateFilePath + ".tmp"
	malformedData := []byte("{\"corrupted_json\": ...")
	if err := os.WriteFile(badTempStateFilePath, malformedData, 0644); err != nil {
		t.Fatalf("Failed to write malformed temp file: %v", err)
	}

	// 3. Attempt to load. Should still load the *original* good state.
	loadedStateAfterBadTemp, err := LoadState(stateFilePath)
	if err != nil {
		t.Fatalf("LoadState() after simulated failed save (bad .tmp) error = %v", err)
	}
	if !reflect.DeepEqual(loadedStateAfterBadTemp, initialState) {
		t.Errorf("LoadState() after simulated failed save (bad .tmp) got = %v, want %v (original state)", loadedStateAfterBadTemp, initialState)
	}
	// Clean up the bad .tmp file
	_ = os.Remove(badTempStateFilePath)

	// Test SaveState creates directory if not exists
	deepDir := filepath.Join(tempDir, "deep", "nested", "dir")
	deepStateFilePath := filepath.Join(deepDir, "deep_state.json")
	newState := &ArchiveProgressState{LastProcessedTopicID: "deep_topic"}

	if err := SaveState(newState, deepStateFilePath); err != nil {
		t.Fatalf("SaveState() to new directory error = %v", err)
	}
	if _, err := os.Stat(deepStateFilePath); os.IsNotExist(err) {
		t.Errorf("SaveState() did not create file in new directory: %s", deepStateFilePath)
	}

	// Test LoadState - malformed JSON file
	malformedJSONFilePath := filepath.Join(tempDir, "malformed.json")
	if err := os.WriteFile(malformedJSONFilePath, []byte("not a json string{"), 0644); err != nil {
		t.Fatalf("Failed to write malformed JSON file: %v", err)
	}
	_, err = LoadState(malformedJSONFilePath)
	if err == nil {
		t.Errorf("LoadState() with malformed JSON expected error, got nil")
	}
}
