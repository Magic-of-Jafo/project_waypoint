package util

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// NormalizeTopicPageURL creates a canonical string representation for a topic page URL.
// It keeps only 'topic', 'forum', and 'start' query parameters.
// It ensures 'forum' matches expectedForumID.
// 'start' is normalized (e.g., "0" or absent for the first page).
// Query parameters are sorted for consistent string output.
func NormalizeTopicPageURL(rawURL string, expectedForumID string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("NormalizeTopicPageURL: failed to parse rawURL '%s': %w", rawURL, err)
	}

	q := u.Query()
	newQ := url.Values{}

	topicID := q.Get("topic")
	if topicID == "" {
		// Fallback: try 't' if 'topic' is not present (common in some forums)
		topicID = q.Get("t")
		if topicID == "" {
			return "", fmt.Errorf("NormalizeTopicPageURL: URL '%s' is missing 'topic' or 't' query parameter", rawURL)
		}
	}
	newQ.Set("topic", topicID)

	// Ensure forum ID is present and correct
	currentForumID := q.Get("forum")
	if currentForumID == "" || currentForumID != expectedForumID {
		// If missing, or different (though less likely if URLs are from same site section),
		// set it to the expected one for consistency.
		newQ.Set("forum", expectedForumID)
	} else {
		newQ.Set("forum", currentForumID) // Use the one from URL if it matches expected
	}

	startVal := q.Get("start")
	if startVal != "" && startVal != "0" {
		newQ.Set("start", startVal)
	} // Else: if start is "0" or empty, omit it for canonical form (first page)

	// Important: To ensure the RawQuery is canonical, we need to sort the keys
	u.RawQuery = encodeSortedQuery(newQ)

	// Clear fragment and user info as they are not relevant for identity here
	u.Fragment = ""
	u.User = nil

	return u.String(), nil
}

// encodeSortedQuery takes url.Values and returns an encoded string with keys sorted.
func encodeSortedQuery(q url.Values) string {
	if q == nil {
		return ""
	}
	var keys []string
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf strings.Builder
	for _, k := range keys {
		vs := q[k]
		keyEscaped := url.QueryEscape(k)
		for _, v := range vs {
			if buf.Len() > 0 {
				buf.WriteByte('&')
			}
			buf.WriteString(keyEscaped)
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
		}
	}
	return buf.String()
}

// TODO: Consider moving the original ParsePaginationLinks, FetchHTML, ExtractTopicsFromHTMLInUtil
// into a different file if htmlutil.go is meant to be lean with only the DefaultHTMLUtil struct methods.
// For now, they are kept here for simplicity as they are used by DefaultHTMLUtil.
