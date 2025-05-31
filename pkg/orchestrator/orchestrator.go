package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// OrchestratorConfig holds the configuration for the extraction orchestrator.
// It defines paths for input data, output data, state management, and logging settings.
type OrchestratorConfig struct {
	TopicListPath  string `json:"topicListPath"`  // Path to the input list of topics (JSON expected)
	ArchivePath    string `json:"archivePath"`    // Path to the root of the Waypoint Archive (used by ProcessTopic)
	OutputJSONPath string `json:"outputJsonPath"` // Path to the directory where JSON files will be saved (used by ProcessTopic)
	StateFilePath  string `json:"stateFilePath"`  // Path to the state file for resumability
	LogLevel       string `json:"logLevel"`       // Logging level (e.g., "DEBUG", "INFO", "WARN", "ERROR")
}

// TopicEntry defines the structure of an entry in the input topic list.
// The input topic list is expected to be a JSON array of these objects.
// Example: [{"topic_id": "123", "subforum_id": "sf45"}, ...]
type TopicEntry struct {
	TopicID    string `json:"topic_id"`
	SubForumID string `json:"subforum_id"` // Kept for now as it's in the input list structure, though not directly used by ProcessTopic call
}

// LoadTopicList reads a JSON file containing a list of topics and their subforum IDs.
// It parses the file into a slice of TopicEntry structs.
func LoadTopicList(filePath string) ([]TopicEntry, error) {
	if filePath == "" {
		return nil, fmt.Errorf("topic list file path cannot be empty")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read topic list file %s: %w", filePath, err)
	}

	if len(data) == 0 {
		// An empty file means no topics to process, which is a valid scenario.
		return []TopicEntry{}, nil
	}

	var topics []TopicEntry
	err = json.Unmarshal(data, &topics)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal topic list JSON from %s: %w", filePath, err)
	}

	// Validate entries: ensure topic_id and subforum_id are present, as they are critical.
	for i, entry := range topics {
		if entry.TopicID == "" {
			return nil, fmt.Errorf("topic entry at index %d in file %s is missing 'topic_id'", i, filePath)
		}
		// SubForumID is defined in TopicEntry, so it should exist if the struct is populated.
		// Validation for its presence can be added if strictness is required beyond struct definition.
		// if entry.SubForumID == "" {
		// 	return nil, fmt.Errorf("topic entry at index %d (topic_id: %s) in file %s is missing 'subforum_id'", i, entry.TopicID, filePath)
		// }
	}

	return topics, nil
}

// State represents the processing status of topics.
// The map key is the topicID, and the value is its status (e.g., "completed", "failed").
type State map[string]string

const (
	// StateCompleted indicates a topic has been successfully processed.
	StateCompleted = "completed"
	// StateFailed indicates a topic processing attempt failed irrevocably.
	StateFailed = "failed"
	// StatePending is not explicitly stored but represents topics not yet in Completed or Failed.
)

// LoadState reads the state file (JSON format) from the given path.
// If the file doesn't exist, it returns an empty state and no error, signifying a fresh run.
func LoadState(stateFilePath string) (State, error) {
	if stateFilePath == "" {
		return nil, fmt.Errorf("state file path cannot be empty")
	}

	data, err := os.ReadFile(stateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found means no previous state, start fresh.
			return make(State), nil
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", stateFilePath, err)
	}

	if len(data) == 0 {
		// Empty file can be treated as a fresh start or an anomaly depending on requirements.
		// For simplicity, treating as a fresh start for now.
		return make(State), nil
	}

	var state State
	err = json.Unmarshal(data, &state)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal state JSON from %s: %w", stateFilePath, err)
	}
	return state, nil
}

// SaveState atomically saves the current state (map of topicID to status) to the given file path as JSON.
// It first writes to a temporary file and then renames it to ensure atomicity.
func SaveState(stateFilePath string, state State) error {
	if stateFilePath == "" {
		return fmt.Errorf("state file path cannot be empty")
	}

	data, err := json.MarshalIndent(state, "", "  ") // Pretty print for readability
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	// Create a temporary file in the same directory to ensure rename is atomic (usually).
	tempFilePath := stateFilePath + ".tmp"

	// Write to the temporary file.
	err = os.WriteFile(tempFilePath, data, 0644) // Standard file permissions
	if err != nil {
		return fmt.Errorf("failed to write state to temporary file %s: %w", tempFilePath, err)
	}

	// Atomically replace the actual state file with the temporary file.
	err = os.Rename(tempFilePath, stateFilePath)
	if err != nil {
		// If rename fails, try to clean up the temp file, but prioritize returning the rename error.
		_ = os.Remove(tempFilePath)
		return fmt.Errorf("failed to rename temporary state file %s to %s: %w", tempFilePath, stateFilePath, err)
	}

	return nil
}

// RunExtractionOrchestrator is the main entry point for the orchestration logic.
// It will manage loading configuration, state, processing topics, and logging.
func RunExtractionOrchestrator(config OrchestratorConfig) error {
	InitLogger(config.LogLevel) // Initialize logger first
	log.Printf("Starting Waypoint Extraction Orchestrator...")
	log.Printf("Configuration: %+v", config)

	// Setup signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	shutdownRequested := false // Flag to indicate if shutdown has been requested

	go func() {
		sig := <-sigCh
		log.Printf("Received signal: %v. Requesting graceful shutdown...", sig)
		shutdownRequested = true
	}()

	// Load the list of all topics to be processed
	log.Println("Loading topic list...")
	allTopics, err := LoadTopicList(config.TopicListPath)
	if err != nil {
		log.Printf("ERROR: Failed to load topic list from %s: %v", config.TopicListPath, err)
		return fmt.Errorf("failed to load topic list: %w", err)
	}
	if len(allTopics) == 0 {
		log.Println("No topics found in the topic list. Orchestrator finished.")
		return nil
	}
	log.Printf("Successfully loaded %d topics from %s", len(allTopics), config.TopicListPath)

	// Load existing processing state
	log.Println("Loading processing state...")
	state, err := LoadState(config.StateFilePath)
	if err != nil {
		log.Printf("CRITICAL ERROR: Failed to load state file from %s: %v. Cannot proceed.", config.StateFilePath, err)
		return fmt.Errorf("failed to load state: %w", err)
	}
	if len(state) > 0 {
		log.Printf("Successfully loaded existing state for %d topics from %s. Will resume.", len(state), config.StateFilePath)
	} else {
		log.Printf("No existing state file found at %s or state file is empty. Starting a fresh run.", config.StateFilePath)
	}

	var topicsProcessedThisRun int
	var topicsFailedThisRun int
	var topicsSkipped int

	log.Printf("Beginning topic processing. Total topics to consider: %d", len(allTopics))
	for i, topicEntry := range allTopics {
		if shutdownRequested {
			log.Printf("Shutdown requested. Interrupting processing before topic ID %s.", topicEntry.TopicID)
			break // Exit the loop
		}

		log.Printf("Processing topic %d/%d: ID=%s, SubForumID=%s", i+1, len(allTopics), topicEntry.TopicID, topicEntry.SubForumID)

		if status, found := state[topicEntry.TopicID]; found && status == StateCompleted {
			log.Printf("Topic %s already marked as '%s'. Skipping.", topicEntry.TopicID, StateCompleted)
			topicsSkipped++
			continue
		} else if found && status == StateFailed {
			log.Printf("Topic %s was previously marked as '%s'. Skipping in this run.", topicEntry.TopicID, StateFailed)
			topicsSkipped++
			continue
		}

		log.Printf("Attempting to process topic ID: %s", topicEntry.TopicID)
		// The actual ProcessTopic function (from topic_processor.go) expects 3 arguments:
		// func ProcessTopic(topicID string, archivePath string, outputPath string) error
		err = ProcessTopic(topicEntry.TopicID, config.ArchivePath, config.OutputJSONPath) // Ensure this is 3 arguments

		if err != nil {
			log.Printf("ERROR: Failed to process topic ID %s: %v", topicEntry.TopicID, err)
			state[topicEntry.TopicID] = StateFailed
			topicsFailedThisRun++
		} else {
			log.Printf("Successfully processed topic ID %s.", topicEntry.TopicID)
			state[topicEntry.TopicID] = StateCompleted
			topicsProcessedThisRun++
		}

		log.Printf("Saving current state after processing topic %s...", topicEntry.TopicID)
		if saveErr := SaveState(config.StateFilePath, state); saveErr != nil {
			log.Printf("CRITICAL ERROR: Failed to save state file to %s after processing topic %s: %v. Subsequent failures might lead to reprocessing.", config.StateFilePath, topicEntry.TopicID, saveErr)
		} else {
			log.Printf("State saved successfully to %s.", config.StateFilePath)
		}
		log.Printf("Progress: %d/%d topics attempted in this run. Current stats - Processed: %d, Failed: %d, Skipped (already done/failed): %d",
			i+1, len(allTopics), topicsProcessedThisRun, topicsFailedThisRun, topicsSkipped)
	}

	if shutdownRequested {
		log.Println("Orchestration run interrupted by signal.")
	} else {
		log.Println("Orchestration run completed normally.")
	}

	log.Printf("Final Summary:")
	log.Printf("  Total topics from input list: %d", len(allTopics))
	log.Printf("  Topics processed successfully in this run: %d", topicsProcessedThisRun)
	log.Printf("  Topics failed to process in this run: %d", topicsFailedThisRun)
	log.Printf("  Topics skipped (previously completed or failed): %d", topicsSkipped)
	log.Printf("  Total topics now marked as '%s' in state: %d", StateCompleted, countStatus(state, StateCompleted))
	log.Printf("  Total topics now marked as '%s' in state: %d", StateFailed, countStatus(state, StateFailed))

	return nil
}

// countStatus is a helper to count topics with a specific status in the state map.
func countStatus(state State, status string) int {
	count := 0
	for _, s := range state {
		if s == status {
			count++
		}
	}
	return count
}
