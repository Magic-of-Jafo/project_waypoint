package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ExtractAuthorUsername extracts the author's username from a post HTML block.
// It returns the username and an error if extraction fails.
func ExtractAuthorUsername(postHTMLBlock *goquery.Document) (string, error) {
	authorCellSelector := "td.normal.bgc1.c.w13.vat"
	authorCell := postHTMLBlock.Find("html body " + authorCellSelector).First()

	if authorCell.Length() == 0 {
		return "", fmt.Errorf("author cell not found with selector: %s (searched from html body)", authorCellSelector)
	}

	strongEl := authorCell.ChildrenFiltered("strong").First()
	if strongEl.Length() == 0 {
		// For debugging, one might want to see the HTML of the cell:
		// cellHTML, _ := authorCell.Html()
		// return "", fmt.Errorf("strong element (username) not found as first strong child within cell selected by '%s'. Cell HTML: %s", authorCellSelector, cellHTML)
		return "", fmt.Errorf("strong element (username) not found as first strong child within cell selected by '%s'", authorCellSelector)
	}

	authorUsername := strings.TrimSpace(strongEl.Text())
	if authorUsername == "" {
		return "", fmt.Errorf("found username strong element but it was empty (selector: %s > strong:first)", authorCellSelector)
	}
	return authorUsername, nil
}

// ExtractTimestamp extracts and parses the post timestamp.
// It returns the timestamp in "YYYY-MM-DD HH:MM:SS" format and an error if extraction or parsing fails.
func ExtractTimestamp(postHTMLBlock *goquery.Document) (string, error) {
	timestampCellSelector := "td.normal.bgc1.vat.w90"
	timestampCell := postHTMLBlock.Find("html body " + timestampCellSelector).First()

	if timestampCell.Length() == 0 {
		return "", fmt.Errorf("timestamp cell not found with selector: %s (searched from html body)", timestampCellSelector)
	}

	// Selector for the span.b from the context of the timestampCell
	targetSpanSelector := "div.vt1.liketext > div.like_left > span.b"
	targetSpan := timestampCell.Find(targetSpanSelector).First()

	if targetSpan.Length() == 0 {
		// For debugging:
		// cellHTML, _ := timestampCell.Html()
		// return "", fmt.Errorf("timestamp span.b element not found with selector '%s' within cell '%s'. Cell HTML: %s", targetSpanSelector, timestampCellSelector, cellHTML)
		return "", fmt.Errorf("timestamp span.b element not found with selector '%s' within cell '%s'", targetSpanSelector, timestampCellSelector)
	}
	rawTimestampStr := targetSpan.Text()
	if rawTimestampStr == "" {
		return "", fmt.Errorf("found timestamp element (span.b) but it was empty (selector: %s > %s)", timestampCellSelector, targetSpanSelector)
	}

	// Normalize non-breaking spaces to regular spaces
	rawTimestampStr = strings.ReplaceAll(rawTimestampStr, "\u00a0", " ")

	// Trim any leading/trailing whitespace
	rawTimestampStr = strings.TrimSpace(rawTimestampStr)

	// Remove exact prefix, including colon and following space
	if strings.HasPrefix(rawTimestampStr, "Posted: ") {
		rawTimestampStr = strings.TrimPrefix(rawTimestampStr, "Posted: ")
	}

	fmt.Printf("DEBUG: rawTimestampStr = %#v\n", rawTimestampStr)

	if rawTimestampStr == "" { // Check after trimming "Posted:"
		return "", fmt.Errorf("timestamp string was empty after trimming 'Posted:' (selector: %s > %s)", timestampCellSelector, targetSpanSelector)
	}

	// Define the expected timestamp layouts. Order matters: try more specific or common ones first.
	// Example: "Jan 23, 2003 02:45 pm"
	// Go's reference time: Mon Jan 2 15:04:05 -0700 MST 2006
	layouts := []string{
		"Jan _2, 2006 03:04 pm", // For "Jan 23, 2003 02:45 pm"
		"Jan _2, 2006 3:04 pm",  // For single digit hour "Jan 23, 2003 2:45 pm"
		// TODO: Add more layouts if other timestamp formats are discovered (e.g., "Today at ...", "Yesterday at ...")
	}

	var t time.Time
	var parseErr error
	parsed := false
	for _, layout := range layouts {
		t, parseErr = time.Parse(layout, rawTimestampStr)
		if parseErr == nil {
			parsed = true
			break
		}
	}

	if !parsed {
		// Subtask 3.5: Log warning for parsing failure (actual logging mechanism to be decided by caller or main app)
		// For now, return a specific error that can be identified and logged by the caller.
		return "", fmt.Errorf("failed to parse timestamp '%s' with known layouts: %w", rawTimestampStr, parseErr) // return last error
	}

	return t.Format("2006-01-02 15:04:05"), nil
}

// ExtractPostID extracts the post ID from a post HTML block.
// It returns the post ID (e.g., "175716" from "p_175716") and an error if extraction fails.
func ExtractPostID(postHTMLBlock *goquery.Document) (string, error) {
	selector := "div.vt1.liketext > div.like_right > span[id^=p_]"
	postIDStr := ""
	var extractionErr error

	postHTMLBlock.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		id, exists := s.Attr("id")
		if !exists {
			extractionErr = fmt.Errorf("found post ID element with selector '%s' but it has no id attribute", selector)
			return false // stop iteration
		}
		if !strings.HasPrefix(id, "p_") {
			extractionErr = fmt.Errorf("found post ID '%s' but it does not start with 'p_'", id)
			return false // stop iteration
		}
		postIDStr = strings.TrimPrefix(id, "p_")
		if postIDStr == "" {
			extractionErr = fmt.Errorf("extracted post ID from attribute '%s' was empty after removing 'p_' prefix", id)
			return false // stop iteration
		}
		return false // Found it, stop iteration
	})

	if extractionErr != nil {
		return "", extractionErr
	}

	if postIDStr == "" {
		return "", fmt.Errorf("post ID element not found with selector: %s, or id attribute was malformed", selector)
	}

	return postIDStr, nil
}

// ExtractPostOrderOnPage extracts the post's order on the page.
// It returns the order (0-indexed) and an error if extraction or parsing fails.
func ExtractPostOrderOnPage(postHTMLBlock *goquery.Document) (int, error) {
	selector := "div.vt1.liketext > div.like_left > span.b > a[name]"
	postOrderStr := ""
	var extractionErr error

	postHTMLBlock.Find(selector).EachWithBreak(func(i int, s *goquery.Selection) bool {
		nameAttr, exists := s.Attr("name")
		if !exists {
			extractionErr = fmt.Errorf("found post order anchor element with selector '%s' but it has no name attribute", selector)
			return false // stop iteration
		}
		postOrderStr = nameAttr
		if postOrderStr == "" {
			extractionErr = fmt.Errorf("extracted post order from name attribute was empty for selector '%s'", selector)
			return false // stop iteration
		}
		return false // Found it, stop iteration
	})

	if extractionErr != nil {
		return 0, extractionErr
	}

	if postOrderStr == "" {
		return 0, fmt.Errorf("post order anchor element not found with selector: %s, or name attribute was missing/empty", selector)
	}

	postOrder, err := strconv.Atoi(postOrderStr)
	if err != nil {
		return 0, fmt.Errorf("failed to convert post order '%s' to integer: %w", postOrderStr, err)
	}

	return postOrder, nil
}
