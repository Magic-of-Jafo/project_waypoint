package data

// Topic represents a single forum topic.
type Topic struct {
	ID         string
	SubForumID string
	Title      string
	URL        string
	// Add other relevant metadata fields here if needed
}

// SubForum represents a sub-forum and its associated topics.
type SubForum struct {
	ID         string
	Name       string
	TopicCount int
	Topics     []Topic // Using a slice of Topic structs
}

// MasterTopicList will hold all topics, ordered and de-duplicated.
// For now, a slice of Topic structs. De-duplication strategy will be
// handled by the logic that populates this.
type MasterTopicList struct {
	Topics []Topic
}
