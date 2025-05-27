package state

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// NewArchiveProgressState creates a new, empty ArchiveProgressState.
func NewArchiveProgressState() *ArchiveProgressState {
	return &ArchiveProgressState{
		ArchivedTopics:     make(map[string]ArchivedTopicDetail),
		JITRefreshAttempts: make(map[string]time.Time),
	}
}

// Save saves the current archival progress to the specified file.
// It uses a safe-writing technique: write to a temporary file first, then rename.
func (aps *ArchiveProgressState) Save(filePath string) error {
	if aps == nil {
		return fmt.Errorf("cannot save a nil ArchiveProgressState")
	}
	// It's good practice to ensure maps are not nil, though NewArchiveProgressState handles this.
	if aps.ArchivedTopics == nil {
		aps.ArchivedTopics = make(map[string]ArchivedTopicDetail)
	}
	if aps.JITRefreshAttempts == nil {
		aps.JITRefreshAttempts = make(map[string]time.Time)
	}

	data, err := json.MarshalIndent(aps, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	tempFilePath := filePath + ".tmp"

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." { // Avoid MkdirAll for current dir or empty string
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for state file %s: %w", dir, err)
		}
	}

	if err := os.WriteFile(tempFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state to temporary file %s: %w", tempFilePath, err)
	}

	if err := os.Rename(tempFilePath, filePath); err != nil {
		return fmt.Errorf("failed to rename temporary state file %s to %s: %w", tempFilePath, filePath, err)
	}

	log.Printf("[INFO] State saved successfully to %s", filePath)
	return nil
}

// LoadState loads the archival progress state from the specified file.
// If the file does not exist, it returns a new, empty state.
// If the file exists but cannot be parsed, it returns an error.
func LoadState(filePath string) (*ArchiveProgressState, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[INFO] State file %s not found. Returning new empty state.", filePath)
			return NewArchiveProgressState(), nil // No state file, start fresh
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", filePath, err)
	}

	var state ArchiveProgressState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state from JSON in file %s: %w", filePath, err)
	}

	// Ensure maps are initialized if they were nil in the JSON (e.g. empty file or old format)
	if state.ArchivedTopics == nil {
		state.ArchivedTopics = make(map[string]ArchivedTopicDetail)
	}
	if state.JITRefreshAttempts == nil {
		state.JITRefreshAttempts = make(map[string]time.Time)
	}

	log.Printf("[INFO] State loaded successfully from %s", filePath)
	return &state, nil
}

// IsTopicArchived checks if a topic is marked as fully archived.
func (aps *ArchiveProgressState) IsTopicArchived(topicID string) bool {
	if aps == nil || aps.ArchivedTopics == nil {
		return false
	}
	_, exists := aps.ArchivedTopics[topicID]
	// For a topic to be considered archived, its entry must exist.
	// Additional checks (e.g., all pages archived) could be added here if TopicDetail had such a flag.
	return exists
}

// MarkTopicAsArchived marks a topic as fully archived.
// This assumes that all its pages have already been marked.
func (aps *ArchiveProgressState) MarkTopicAsArchived(topicID string) {
	if aps == nil {
		log.Println("[ERROR] MarkTopicAsArchived called on nil ArchiveProgressState")
		return
	}
	if aps.ArchivedTopics == nil {
		aps.ArchivedTopics = make(map[string]ArchivedTopicDetail)
	}

	detail, exists := aps.ArchivedTopics[topicID]
	if !exists {
		// If topic detail doesn't exist, create it.
		// This might happen if pages are archived before the topic itself is explicitly marked.
		detail = ArchivedTopicDetail{
			TopicID:       topicID,
			ArchivedAt:    time.Now().UTC(),
			ArchivedPages: make(map[int]ArchivedPageDetail),
		}
	} else {
		// If it exists, just update its ArchivedAt timestamp
		detail.ArchivedAt = time.Now().UTC()
	}
	aps.ArchivedTopics[topicID] = detail
}

// IsPageArchived checks if a specific page of a topic is archived.
func (aps *ArchiveProgressState) IsPageArchived(topicID string, pageNum int) bool {
	if aps == nil || aps.ArchivedTopics == nil {
		return false
	}
	topicDetail, topicExists := aps.ArchivedTopics[topicID]
	if !topicExists || topicDetail.ArchivedPages == nil {
		return false
	}
	_, pageExists := topicDetail.ArchivedPages[pageNum]
	return pageExists
}

// MarkPageAsArchived marks a specific page of a topic as archived.
func (aps *ArchiveProgressState) MarkPageAsArchived(topicID string, pageNum int, pageURL string) {
	if aps == nil {
		log.Println("[ERROR] MarkPageAsArchived called on nil ArchiveProgressState")
		return
	}
	if aps.ArchivedTopics == nil {
		aps.ArchivedTopics = make(map[string]ArchivedTopicDetail)
	}

	topicDetail, exists := aps.ArchivedTopics[topicID]
	if !exists {
		topicDetail = ArchivedTopicDetail{
			TopicID: topicID,
			// ArchivedAt for the topic itself is marked when the topic is fully done.
			ArchivedPages: make(map[int]ArchivedPageDetail),
		}
	}
	if topicDetail.ArchivedPages == nil { // Should be handled by above, but defensive check
		topicDetail.ArchivedPages = make(map[int]ArchivedPageDetail)
	}

	topicDetail.ArchivedPages[pageNum] = ArchivedPageDetail{URL: pageURL /* ArchivedAt: time.Now().UTC() */}
	aps.ArchivedTopics[topicID] = topicDetail
}

// MarkJITRefreshAttempted records the time a JIT refresh was attempted for a sub-forum.
func (aps *ArchiveProgressState) MarkJITRefreshAttempted(subForumID string, attemptTime time.Time) {
	if aps == nil {
		log.Println("[ERROR] MarkJITRefreshAttempted called on nil ArchiveProgressState")
		return
	}
	if aps.JITRefreshAttempts == nil {
		aps.JITRefreshAttempts = make(map[string]time.Time)
	}
	aps.JITRefreshAttempts[subForumID] = attemptTime
}

// TotalPagesArchived counts the total number of pages archived across all topics.
func (aps *ArchiveProgressState) TotalPagesArchived() int {
	if aps == nil || aps.ArchivedTopics == nil {
		return 0
	}
	count := 0
	for _, topicDetail := range aps.ArchivedTopics {
		if topicDetail.ArchivedPages != nil {
			count += len(topicDetail.ArchivedPages)
		}
	}
	return count
}

// SaveProgress is a package-level function that saves the global CurrentState.
func SaveProgress(filePath string) error {
	if CurrentState == nil {
		log.Println("[WARNING] SaveProgress called but CurrentState is nil. Nothing to save.")
		// Depending on desired behavior, could return an error or just log and return nil.
		// For now, let's prevent a panic and allow the archiver to decide if this is fatal.
		return fmt.Errorf("CurrentState is nil, cannot save progress")
	}
	return CurrentState.Save(filePath)
}
