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
	ID         string  `json:"sub_forum_id"` // Changed from int to string
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

// PostMetadata holds the extracted metadata for a single forum post.
type PostMetadata struct {
	PostID          string `json:"post_id"`
	TopicID         string `json:"topic_id"`
	SubForumID      string `json:"subforum_id"`
	PageNumber      int    `json:"page_number"`
	PostOrderOnPage int    `json:"post_order_on_page"`
	AuthorUsername  string `json:"author_username"`
	Timestamp       string `json:"timestamp"` // Formatted "YYYY-MM-DD HH:MM:SS"
}
