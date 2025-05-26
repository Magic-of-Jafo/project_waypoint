package state

// ArchiveProgressState holds the state of the archival process.
type ArchiveProgressState struct {
	LastProcessedSubForumID            string   `json:"last_processed_sub_forum_id"`
	LastProcessedTopicID               string   `json:"last_processed_topic_id"`
	LastProcessedPageNumberInTopic     int      `json:"last_processed_page_number_in_topic"`
	ProcessedTopicIDsInCurrentSubForum []string `json:"processed_topic_ids_in_current_sub_forum"`
	CompletedSubForumIDs               []string `json:"completed_sub_forum_ids"`
}
