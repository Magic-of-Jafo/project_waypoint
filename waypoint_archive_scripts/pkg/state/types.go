package state

import "time"

// ArchivedPageDetail stores information about an archived page.
// Currently, just the URL, but could be extended (e.g., timestamp, size).
type ArchivedPageDetail struct {
	URL string `json:"url"`
	// ArchivedAt time.Time `json:"archived_at"` // Example extension
}

// ArchivedTopicDetail holds information about an archived topic, including its pages.
type ArchivedTopicDetail struct {
	TopicID       string                     `json:"topic_id"`
	ArchivedAt    time.Time                  `json:"archived_at"`
	ArchivedPages map[int]ArchivedPageDetail `json:"archived_pages"` // Page number to PageDetail
}

// ArchiveProgressState holds the overall state of the archival process.
type ArchiveProgressState struct {
	ArchivedTopics     map[string]ArchivedTopicDetail `json:"archived_topics"`      // TopicID to ArchivedTopicDetail
	JITRefreshAttempts map[string]time.Time           `json:"jit_refresh_attempts"` // SubForumID to last attempt time

	// Fields for resuming progress
	LastProcessedSubForumID            string   `json:"last_processed_sub_forum_id"`
	LastProcessedTopicID               string   `json:"last_processed_topic_id"`
	LastProcessedPageNumberInTopic     int      `json:"last_processed_page_number_in_topic"`
	ProcessedTopicIDsInCurrentSubForum []string `json:"processed_topic_ids_in_current_sub_forum"`
	CompletedSubForumIDs               []string `json:"completed_sub_forum_ids"`

	// Other global state fields can be added here if needed,
	// e.g., LastSuccessfulFullRun time.Time
}

// CurrentState is the global instance of the archival progress.
// It is loaded at startup and saved periodically.
var CurrentState *ArchiveProgressState
