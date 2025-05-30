package state

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestSaveAndLoadState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFilePath := filepath.Join(tempDir, "test_archive_progress.json")
	tempStateFilePath := stateFilePath + ".tmp"

	initialState := NewArchiveProgressState()
	initialState.LastProcessedSubForumID = "subforum1"
	initialState.LastProcessedTopicID = "topic123"
	initialState.LastProcessedPageNumberInTopic = 5
	initialState.ProcessedTopicIDsInCurrentSubForum = []string{"topic100", "topic101"}
	initialState.CompletedSubForumIDs = []string{"subforum_A", "subforum_B"}

	if err := initialState.Save(stateFilePath); err != nil {
		t.Fatalf("initialState.Save() error = %v", err)
	}

	if _, err := os.Stat(tempStateFilePath); !os.IsNotExist(err) {
		t.Errorf("Temporary state file %s was not removed after saving", tempStateFilePath)
	}

	loadedState, err := LoadState(stateFilePath)
	if err != nil {
		t.Fatalf("LoadState() error = %v", err)
	}
	if loadedState.ArchivedTopics == nil {
		loadedState.ArchivedTopics = make(map[string]ArchivedTopicDetail)
	}
	if loadedState.JITRefreshAttempts == nil {
		loadedState.JITRefreshAttempts = make(map[string]time.Time)
	}
	if initialState.ArchivedTopics == nil {
		initialState.ArchivedTopics = make(map[string]ArchivedTopicDetail)
	}
	if initialState.JITRefreshAttempts == nil {
		initialState.JITRefreshAttempts = make(map[string]time.Time)
	}

	if !reflect.DeepEqual(loadedState, initialState) {
		t.Errorf("LoadState() got = %+v, want %+v", loadedState, initialState)
	}

	nonExistentFilePath := filepath.Join(tempDir, "does_not_exist.json")
	state, err := LoadState(nonExistentFilePath)
	if err != nil {
		t.Errorf("LoadState() for non-existent file error = %v, want nil", err)
	}
	expectedEmptyState := NewArchiveProgressState()
	if !reflect.DeepEqual(state, expectedEmptyState) {
		t.Errorf("LoadState() for non-existent file got state = %+v, want new empty state %+v", state, expectedEmptyState)
	}

	badTempStateFilePath := stateFilePath + ".tmp"
	malformedData := []byte("{\"corrupted_json\": ...")
	if err := os.WriteFile(badTempStateFilePath, malformedData, 0644); err != nil {
		t.Fatalf("Failed to write malformed temp file: %v", err)
	}

	loadedStateAfterBadTemp, err := LoadState(stateFilePath)
	if err != nil {
		t.Fatalf("LoadState() after simulated failed save (bad .tmp) error = %v", err)
	}
	if !reflect.DeepEqual(loadedStateAfterBadTemp, initialState) {
		t.Errorf("LoadState() after simulated failed save (bad .tmp) got = %+v, want %+v (original state)", loadedStateAfterBadTemp, initialState)
	}
	_ = os.Remove(badTempStateFilePath)

	deepDir := filepath.Join(tempDir, "deep", "nested", "dir")
	deepStateFilePath := filepath.Join(deepDir, "deep_state.json")
	newState := NewArchiveProgressState()
	newState.LastProcessedTopicID = "deep_topic"

	if err := newState.Save(deepStateFilePath); err != nil {
		t.Fatalf("newState.Save() to new directory error = %v", err)
	}
	if _, err := os.Stat(deepStateFilePath); os.IsNotExist(err) {
		t.Errorf("newState.Save() did not create file in new directory: %s", deepStateFilePath)
	}

	malformedJSONFilePath := filepath.Join(tempDir, "malformed.json")
	if err := os.WriteFile(malformedJSONFilePath, []byte("not a json string{"), 0644); err != nil {
		t.Fatalf("Failed to write malformed JSON file: %v", err)
	}
	_, err = LoadState(malformedJSONFilePath)
	if err == nil {
		t.Errorf("LoadState() with malformed JSON expected error, got nil")
	}
}
