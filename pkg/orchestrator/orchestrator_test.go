package orchestrator

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSaveAndLoadState(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFilePath := filepath.Join(tempDir, "test_state.json")

	// 1. Test LoadState: Non-existent file
	t.Run("LoadState_NonExistentFile", func(t *testing.T) {
		loadedState, err := LoadState(stateFilePath) // File doesn't exist yet
		if err != nil {
			t.Errorf("LoadState with non-existent file returned error: %v", err)
		}
		if len(loadedState) != 0 {
			t.Errorf("LoadState with non-existent file expected empty state, got %v", loadedState)
		}
	})

	// 2. Test SaveState: Normal save
	t.Run("SaveState_Normal", func(t *testing.T) {
		expectedState := State{
			"topic1": StateCompleted,
			"topic2": StateFailed,
			"topic3": StateCompleted,
		}
		err := SaveState(stateFilePath, expectedState)
		if err != nil {
			t.Fatalf("SaveState failed: %v", err)
		}

		// Verify file content directly for robustness
		data, readErr := os.ReadFile(stateFilePath)
		if readErr != nil {
			t.Fatalf("Failed to read back state file: %v", readErr)
		}
		var actualStateFromFile State
		if unmarshalErr := json.Unmarshal(data, &actualStateFromFile); unmarshalErr != nil {
			t.Fatalf("Failed to unmarshal saved state from file: %v", unmarshalErr)
		}
		if !reflect.DeepEqual(expectedState, actualStateFromFile) {
			t.Errorf("Saved state in file an_orchestrator_test.go %v, want %v", actualStateFromFile, expectedState)
		}
	})

	// 3. Test LoadState: Normal load of previously saved state
	t.Run("LoadState_Normal", func(t *testing.T) {
		// State file should exist from previous sub-test
		loadedState, err := LoadState(stateFilePath)
		if err != nil {
			t.Fatalf("LoadState failed: %v", err)
		}
		expectedState := State{
			"topic1": StateCompleted,
			"topic2": StateFailed,
			"topic3": StateCompleted,
		}
		if !reflect.DeepEqual(expectedState, loadedState) {
			t.Errorf("LoadState loaded %v, want %v", loadedState, expectedState)
		}
	})

	// 4. Test LoadState: Empty state file
	t.Run("LoadState_EmptyFile", func(t *testing.T) {
		emptyFilePath := filepath.Join(tempDir, "empty_state.json")
		if err := os.WriteFile(emptyFilePath, []byte{}, 0644); err != nil {
			t.Fatalf("Failed to create empty state file: %v", err)
		}
		loadedState, err := LoadState(emptyFilePath)
		if err != nil {
			t.Errorf("LoadState with empty file returned error: %v", err)
		}
		if len(loadedState) != 0 {
			t.Errorf("LoadState with empty file expected empty state, got %v", loadedState)
		}
	})

	// 5. Test LoadState: Corrupted JSON file
	t.Run("LoadState_CorruptedJSON", func(t *testing.T) {
		corruptedFilePath := filepath.Join(tempDir, "corrupted_state.json")
		corruptedJSON := []byte(`{"topic1": "completed", "topic2": "failed"`) // Missing closing brace
		if err := os.WriteFile(corruptedFilePath, corruptedJSON, 0644); err != nil {
			t.Fatalf("Failed to write corrupted state file: %v", err)
		}
		_, err := LoadState(corruptedFilePath)
		if err == nil {
			t.Errorf("LoadState with corrupted JSON expected an error, got nil")
		}
	})

	// 6. Test SaveState: Ensure .tmp file is removed after successful save
	t.Run("SaveState_TempFileCleanup", func(t *testing.T) {
		pathForCleanupTest := filepath.Join(tempDir, "cleanup_state.json")
		tempPathForCleanupTest := pathForCleanupTest + ".tmp"
		stateToSave := State{"cleanup1": StateCompleted}

		// Ensure no temp file exists before save (it shouldn't from previous tests)
		if _, err := os.Stat(tempPathForCleanupTest); !os.IsNotExist(err) {
			t.Fatalf("Temp file %s exists before SaveState, it should not.", tempPathForCleanupTest)
		}

		err := SaveState(pathForCleanupTest, stateToSave)
		if err != nil {
			t.Fatalf("SaveState failed for cleanup test: %v", err)
		}

		// Check main file exists
		if _, err := os.Stat(pathForCleanupTest); os.IsNotExist(err) {
			t.Errorf("Main state file %s does not exist after successful save.", pathForCleanupTest)
		}
		// Check temp file does NOT exist
		if _, err := os.Stat(tempPathForCleanupTest); !os.IsNotExist(err) {
			t.Errorf("Temp file %s was not removed after successful SaveState.", tempPathForCleanupTest)
		}
	})

	// 7. Test SaveState and LoadState with empty state map
	t.Run("SaveAndLoad_EmptyStateMap", func(t *testing.T) {
		emptyStatePath := filepath.Join(tempDir, "empty_map_state.json")
		emptyState := make(State)
		err := SaveState(emptyStatePath, emptyState)
		if err != nil {
			t.Fatalf("SaveState with empty map failed: %v", err)
		}

		loadedState, err := LoadState(emptyStatePath)
		if err != nil {
			t.Errorf("LoadState failed for empty map state file: %v", err)
		}
		if len(loadedState) != 0 {
			t.Errorf("LoadState from saved empty map: expected empty state, got %v", loadedState)
		}
	})
}

// mockProcessTopic can be used by specific tests if they directly call a helper that uses it.
// However, RunExtractionOrchestrator calls the *real* ProcessTopic from topic_processor.go.
// So, assignments to this variable within TestRunExtractionOrchestrator will not mock the
// ProcessTopic calls made by RunExtractionOrchestrator itself.
var mockProcessTopic func(topicID string, archivePath string, outputPath string) error

func TestRunExtractionOrchestrator(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "orchestrator_run_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir for orchestrator run: %v", err)
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "archive")
	outputPath := filepath.Join(tempDir, "output")
	topicListPath := filepath.Join(tempDir, "topics.json")
	stateFilePath := filepath.Join(tempDir, "run_state.json")

	if err := os.MkdirAll(archivePath, 0755); err != nil {
		t.Fatalf("Failed to create archive dir: %v", err)
	}
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		t.Fatalf("Failed to create output dir: %v", err)
	}

	// Helper to write topic list file
	writeTopicList := func(path string, topics []TopicEntry) {
		data, _ := json.MarshalIndent(topics, "", "  ")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to write topic list file %s: %v", path, err)
		}
	}

	// Helper to write state file
	writeStateFile := func(path string, state State) {
		data, _ := json.MarshalIndent(state, "", "  ")
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to write state file %s: %v", path, err)
		}
	}

	baseConfig := OrchestratorConfig{
		TopicListPath:  topicListPath,
		ArchivePath:    archivePath,
		OutputJSONPath: outputPath,
		StateFilePath:  stateFilePath,
		LogLevel:       "DEBUG", // Use DEBUG for tests to see more log output
	}

	t.Run("FreshRun_AllSuccess", func(t *testing.T) {
		// Setup: Create a topic list, no initial state file
		topics := []TopicEntry{
			{TopicID: "t1", SubForumID: "sf1"},
			{TopicID: "t2", SubForumID: "sf1"},
		}
		writeTopicList(topicListPath, topics)
		_ = os.Remove(stateFilePath) // Ensure no state file exists

		// This mock assignment will NOT affect the ProcessTopic called by RunExtractionOrchestrator
		// as it calls the real one from topic_processor.go. Test relies on real implementation.
		// For this test to pass, the real ProcessTopic needs to behave as expected (e.g. create dummy files or be robust to missing ones).
		// We will need to setup mock archive files.
		setupMockArchiveForTopics(t, archivePath, topics)

		// This var is not used by the real ProcessTopic
		// processedTopics := make(map[string]int) // Commented out as it's not useful here
		// mockProcessTopic = func(topicID, _, _ string) error {
		// 	processedTopics[topicID]++
		// 	return nil
		// }

		err := RunExtractionOrchestrator(baseConfig)
		if err != nil {
			t.Errorf("RunExtractionOrchestrator failed: %v", err)
		}

		// We need to check side effects of the *real* ProcessTopic, e.g. created files or state.
		// The processedTopics map above won't be populated by the real execution path.
		// So, we verify by checking the final state file.

		finalState, loadErr := LoadState(stateFilePath)
		if loadErr != nil {
			t.Fatalf("Failed to load final state: %v", loadErr)
		}
		expectedState := State{"t1": StateCompleted, "t2": StateCompleted}
		if !reflect.DeepEqual(expectedState, finalState) {
			t.Errorf("Final state an_orchestrator_test.go %v, want %v", finalState, expectedState)
		}
	})

	t.Run("ResumeRun_PartialSuccessAndFailure", func(t *testing.T) {
		// Setup: Topic list, initial state with t1 completed, t2 failed
		topics := []TopicEntry{
			{TopicID: "t1", SubForumID: "sf1"}, // Should be skipped
			{TopicID: "t2", SubForumID: "sf1"}, // Should be skipped (as failed)
			{TopicID: "t3", SubForumID: "sf2"}, // Should be processed (success)
			{TopicID: "t4", SubForumID: "sf2"}, // Should be processed (fail)
		}
		writeTopicList(topicListPath, topics)
		initialState := State{"t1": StateCompleted, "t2": StateFailed}
		writeStateFile(stateFilePath, initialState)

		// Setup mock archive files for t3 and t4.
		// For t3 (success), create its files.
		setupMockArchiveForTopics(t, archivePath, []TopicEntry{topics[2]})

		// For t4 (failure), ensure its directory is missing or its specific page is missing to cause ProcessTopic to error.
		topicDirT4 := getTestSubforumDir(archivePath, topics[3].SubForumID, topics[3].TopicID)
		_ = os.RemoveAll(topicDirT4) // Ensure t4 will fail to find files for ProcessTopic

		err := RunExtractionOrchestrator(baseConfig)
		if err != nil {
			t.Errorf("RunExtractionOrchestrator failed: %v", err)
		}

		finalState, _ := LoadState(stateFilePath)
		expectedState := State{
			"t1": StateCompleted, // From initial
			"t2": StateFailed,    // From initial
			"t3": StateCompleted, // Processed successfully
			"t4": StateFailed,    // Processed with failure (due to missing files)
		}
		if !reflect.DeepEqual(expectedState, finalState) {
			t.Errorf("Final state an_orchestrator_test.go %v, want %v", finalState, expectedState)
		}
	})

	t.Run("EmptyTopicList", func(t *testing.T) {
		writeTopicList(topicListPath, []TopicEntry{})
		_ = os.Remove(stateFilePath)

		err := RunExtractionOrchestrator(baseConfig)
		if err != nil {
			t.Errorf("RunExtractionOrchestrator with empty list failed: %v", err)
		}
		// State file might be created empty or not at all, either is fine.
		// If it's created, it should be empty.
		if _, statErr := os.Stat(stateFilePath); !os.IsNotExist(statErr) {
			finalState, _ := LoadState(stateFilePath)
			if len(finalState) != 0 {
				t.Errorf("State file for empty run should be empty, got %v", finalState)
			}
		}
	})

	t.Run("TopicListLoadFail", func(t *testing.T) {
		_ = os.Remove(topicListPath) // Make LoadTopicList fail
		_ = os.Remove(stateFilePath)

		configWithBadTopicPath := baseConfig

		err := RunExtractionOrchestrator(configWithBadTopicPath)
		if err == nil {
			t.Errorf("RunExtractionOrchestrator expected error with bad topic list path, got nil")
		}
	})

	t.Run("StateFileLoadFail_NotCriticalForOrchestratorFunctionItself", func(t *testing.T) {
		topics := []TopicEntry{{TopicID: "tfail1", SubForumID: "sff1"}}
		writeTopicList(topicListPath, topics)
		_ = os.Remove(stateFilePath) // Ensure no state file for fresh run

		setupMockArchiveForTopics(t, archivePath, topics) // Setup for tfail1 to succeed

		err := RunExtractionOrchestrator(baseConfig)
		if err != nil {
			t.Errorf("RunExtractionOrchestrator with no state file (fresh run) returned error: %v", err)
		}
		finalState, _ := LoadState(stateFilePath)
		if finalState["tfail1"] != StateCompleted {
			t.Errorf("Expected tfail1 to be completed in state, got %v", finalState)
		}
	})
}

// Helper function to create mock HTML files for topics for integration testing
func setupMockArchiveForTopics(t *testing.T, archiveBasePath string, topics []TopicEntry) {
	t.Helper()
	for _, topic := range topics {
		subForumDir := filepath.Join(archiveBasePath, topic.SubForumID)
		topicDir := filepath.Join(subForumDir, topic.TopicID)

		if err := os.MkdirAll(topicDir, 0755); err != nil {
			t.Fatalf("setupMockArchiveForTopics: failed to create directory %s: %v", topicDir, err)
		}
		// Create a simple page_1.html for successful processing by the real ProcessTopic
		// The real ProcessTopic expects a certain HTML structure to find post blocks.
		// Corrected raw string literal:
		mockHTMLContent := `
<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
    <div id="container">
        <table class="normal"><tr><td>Navigation Table, to be skipped</td></tr></table>
        <table class="normal"> <!-- This is the postsTable -->
            <tr> <!-- Post 1 -->
                <td class="normal bgc1 c w13 vat"><strong>Author1</strong></td>
                <td class="normal bgc1 vat w90">
                    <div class="vt1 liketext">
                        <div class="like_left">Posted: <span class="b">Jan 01, 2023 10:00 am</span><a name="0"></a></div>
                        <div class="like_right"><span id="p_mockpost123"></span></div>
                    </div>
                    <hr />
                    <div>Post content 1</div>
                </td>
            </tr>
        </table>
    </div>
</body>
</html>
`
		pagePath := filepath.Join(topicDir, "page_1.html")
		if err := os.WriteFile(pagePath, []byte(mockHTMLContent), 0644); err != nil {
			t.Fatalf("setupMockArchiveForTopics: failed to write mock HTML file %s: %v", pagePath, err)
		}
		log.Printf("TestSetup: Created mock file %s", pagePath)
	}
}

// getTestSubforumDir is a helper for creating unique directory paths for test files,
// ensuring that the subforum ID is part of the path as ProcessTopic might expect.
func getSubforumPathHelper(baseArchivePath string, testName string, subforumID string, topicID string) string {
	return filepath.Join(baseArchivePath, subforumID, topicID)
}
