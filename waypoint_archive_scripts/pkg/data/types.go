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
	ID         int     `json:"sub_forum_id"`
	Name       string  `json:"sub_forum_name"`
	URL        string  `json:"base_url"` // Temporary tag
	TopicCount int     `json:"topics_count"`
	Topics     []Topic `json:"-"` // Topics are not in subforum_list.json, explicitly ignore
}

// MasterTopicList will hold all topics, ordered and de-duplicated.
// For now, a slice of Topic structs. De-duplication strategy will be
// handled by the logic that populates this.
type MasterTopicList struct {
	Topics []Topic
}
