package data

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
