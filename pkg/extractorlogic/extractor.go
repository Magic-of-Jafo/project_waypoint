package extractorlogic

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"pkg/data"
	"pkg/parser"

	"github.com/PuerkitoBio/goquery"
)

// ExtractPostMetadata extracts all core metadata from a post HTML block and its file path.
func ExtractPostMetadata(postHTMLBlock *goquery.Document, filePath string) (data.PostMetadata, error) {
	var metadata data.PostMetadata
	var err error
	var errs []string // To collect multiple errors

	// --- Contextual Metadata from filePath (Subtasks 6.2, 6.3) ---
	cleanPath := filepath.ToSlash(filePath)
	parts := strings.Split(cleanPath, "/")

	if len(parts) < 3 {
		return metadata, fmt.Errorf("filePath '%s' is too short to extract subforum, topic, and page number", filePath)
	}
	pageFileName := parts[len(parts)-1]
	metadata.TopicID = parts[len(parts)-2]
	metadata.SubForumID = parts[len(parts)-3]
	if strings.HasPrefix(pageFileName, "page_") && strings.HasSuffix(pageFileName, ".html") {
		numStr := strings.TrimSuffix(strings.TrimPrefix(pageFileName, "page_"), ".html")
		metadata.PageNumber, err = strconv.Atoi(numStr)
		if err != nil {
			err = fmt.Errorf("failed to parse page number from '%s': %w", pageFileName, err)
			errs = append(errs, err.Error()) // Collect error, continue if possible
		}
	} else {
		err = fmt.Errorf("page filename '%s' does not match expected format 'page_{number}.html'", pageFileName)
		errs = append(errs, err.Error())
	}

	// --- Metadata from HTML content (Subtask 7.1) ---

	// AuthorUsername (Task 2)
	metadata.AuthorUsername, err = parser.ExtractAuthorUsername(postHTMLBlock)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to extract author username: %s", err.Error()))
		// AC8: Log warning/error but attempt to extract as much as possible.
		// Depending on logging strategy, this could be logged here or by the caller.
	}

	// Timestamp (Task 3)
	metadata.Timestamp, err = parser.ExtractTimestamp(postHTMLBlock)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to extract timestamp: %s", err.Error()))
	}

	// PostID (Task 4)
	metadata.PostID, err = parser.ExtractPostID(postHTMLBlock)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to extract post ID: %s", err.Error()))
	}

	// PostOrderOnPage (Task 5)
	metadata.PostOrderOnPage, err = parser.ExtractPostOrderOnPage(postHTMLBlock)
	if err != nil {
		errs = append(errs, fmt.Sprintf("failed to extract post order on page: %s", err.Error()))
	}

	// Subtask 7.2: Comprehensive error handling and aggregation
	if len(errs) > 0 {
		// Return the partially populated metadata along with the collected errors.
		// The caller can decide how to handle/log these.
		// AC8 compliance: Attempt to extract as much as possible, mark post as having issues.
		return metadata, fmt.Errorf("encountered %d error(s) during metadata extraction:\n%s", len(errs), strings.Join(errs, "\n"))
	}

	return metadata, nil
}
