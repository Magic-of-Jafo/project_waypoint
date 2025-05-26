package data

// Topic represents a single forum topic.
type Topic struct {
	ID                   string
	SubForumID           string
	Title                string
	URL                  string
	AuthorUsername       string
	Replies              int
	Views                int
	LastPostUsername     string
	LastPostTimestampRaw string // Raw timestamp string from CSV
	IsSticky             bool
	IsLocked             bool
	// Add other relevant metadata fields here if needed
}

// SubForum represents a sub-forum and its associated topics.
type SubForum struct {
	ID         string
	Name       string
	URL        string // Added for Story 2.8 - JIT Refresh. Base URL for the sub-forum page.
	TopicCount int
	Topics     []Topic // Using a slice of Topic structs
}

// MasterTopicList will hold all topics, ordered and de-duplicated.
// For now, a slice of Topic structs. De-duplication strategy will be
// handled by the logic that populates this.
type MasterTopicList struct {
	Topics []Topic
}
