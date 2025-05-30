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

// ContentBlockType defines the type of content block.
type ContentBlockType string

const (
	// ContentBlockTypeNewText represents a block of new text from the author.
	ContentBlockTypeNewText ContentBlockType = "new_text"
	// ContentBlockTypeQuote represents a block of quoted text.
	ContentBlockTypeQuote ContentBlockType = "quote"
)

// ContentBlock represents a block of content within a post, which can be
// either new text from the author or a quote.
type ContentBlock struct {
	Type            ContentBlockType `json:"type"`
	Content         string           `json:"content,omitempty"`          // Used for new_text
	QuotedUser      string           `json:"quoted_user,omitempty"`      // Used for quote
	QuotedTimestamp string           `json:"quoted_timestamp,omitempty"` // Used for quote, nullable
	QuotedText      string           `json:"quoted_text,omitempty"`      // Used for quote
}
