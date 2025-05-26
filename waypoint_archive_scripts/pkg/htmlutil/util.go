package htmlutil

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html/charset"

	"waypoint_archive_scripts/pkg/data"
)

// FetchHTML retrieves the HTML content from a given URL with a politeness delay.
// It attempts to handle character encoding.
func FetchHTML(pageURL string, delay time.Duration, userAgent string) (string, error) {
	if delay > 0 {
		log.Printf("[DEBUG] FetchHTML: Applying politeness delay of %v for URL: %s", delay, pageURL)
		time.Sleep(delay)
	}

	log.Printf("[DEBUG] FetchHTML: Fetching URL: %s", pageURL)
	client := &http.Client{
		Timeout: 30 * time.Second, // Reasonable timeout
	}
	req, err := http.NewRequest("GET", pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("FetchHTML: failed to create request for %s: %w", pageURL, err)
	}
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", "WaypointArchiveAgent/1.0 (htmlutil)") // Default if none provided
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("FetchHTML: failed to get URL %s: %w", pageURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("FetchHTML: request to %s failed with status %s", pageURL, resp.Status)
	}

	contentType := resp.Header.Get("Content-Type")
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("FetchHTML: failed to read response body from %s: %w", pageURL, err)
	}

	// Attempt to determine encoding
	var reader io.Reader = bytes.NewReader(bodyBytes)
	utf8Reader, err := charset.NewReader(reader, contentType)
	if err == nil {
		reader = utf8Reader
	} else {
		log.Printf("[WARNING] FetchHTML: Could not determine charset for %s (Content-Type: %s): %v. Falling back to raw bytes.", pageURL, contentType, err)
		// Fallback to original reader (bodyBytes) if charset detection fails
		reader = bytes.NewReader(bodyBytes)
	}

	// Read the potentially transformed body
	utf8Bytes, err := io.ReadAll(reader)
	if err != nil {
		// If reading from the transforming reader fails, try reading raw bodyBytes as a last resort
		log.Printf("[WARNING] FetchHTML: Error reading transformed content for %s: %v. Trying raw bytes.", pageURL, err)
		return string(bodyBytes), nil // Return raw bytes as string
	}

	log.Printf("[DEBUG] FetchHTML: Successfully fetched and decoded URL: %s (Size: %d bytes)", pageURL, len(utf8Bytes))
	return string(utf8Bytes), nil
}

// ParsePaginationLinks extracts all unique pagination links from HTML content.
// It assumes pagination links are within a common structure (e.g., div.pagination a).
func ParsePaginationLinks(htmlContent string, pageURL string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("ParsePaginationLinks: failed to create goquery document for %s: %w", pageURL, err)
	}

	parsedPageURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("ParsePaginationLinks: failed to parse page URL %s: %w", pageURL, err)
	}

	var links []string
	seenLinks := make(map[string]bool)

	// Common pagination selectors - this might need to be made more configurable or robust
	// Based on the previous JIT refresh logic, it was looking for "div.pagination a"
	doc.Find("div.pagination a[href], .pagmenu a[href], .page-nav a[href], .nav-links a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" || href == "#" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return
		}

		absURL, err := parsedPageURL.Parse(href)
		if err != nil {
			log.Printf("[WARNING] ParsePaginationLinks: Error parsing pagination link '%s' on page %s: %v", href, pageURL, err)
			return
		}
		absLinkStr := absURL.String()
		if !seenLinks[absLinkStr] {
			links = append(links, absLinkStr)
			seenLinks[absLinkStr] = true
		}
	})

	if len(links) == 0 {
		log.Printf("[DEBUG] ParsePaginationLinks: No pagination links found on %s using common selectors.", pageURL)
	} else {
		log.Printf("[DEBUG] ParsePaginationLinks: Found %d unique pagination links on %s.", len(links), pageURL)
	}

	return links, nil
}

// ExtractTopicsFromHTMLInUtil parses the HTML content of a sub-forum page and extracts topics.
// This is adapted from internal/indexer/topic/ExtractTopics.
func ExtractTopicsFromHTMLInUtil(htmlContent string, pageURL string, subForumID string) ([]data.Topic, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("ExtractTopicsFromHTMLInUtil: failed to create goquery document for %s: %w", pageURL, err)
	}

	parsedPageURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("ExtractTopicsFromHTMLInUtil: failed to parse page URL %s: %w", pageURL, err)
	}

	var topics []data.Topic = make([]data.Topic, 0)
	// Note: The original ExtractTopics had a seenTopicIDs map for on-page de-duplication.
	// This is removed here as JIT refresh logic handles de-duplication at a higher level
	// (against existing index and across multiple JIT pages).
	// If on-page duplicates are possible from the source HTML structure, it could be re-added.

	// Selector based on the provided internal/indexer/topic/topic.go
	// "table.normal tr" and then "td.normal.bgc2 > a.b[href*='viewtopic.php']"
	doc.Find("table.normal tr").Each(func(i int, tr *goquery.Selection) {
		tr.Find("td.normal.bgc2 > a.b[href*='viewtopic.php'], a.topic-title[href*='viewtopic.php'], a[href*='viewtopic.php'][title*='Topic:']").Each(func(j int, link *goquery.Selection) {
			href, exists := link.Attr("href")
			if !exists {
				return
			}

			topicTitle := strings.TrimSpace(link.Text())
			if topicTitle == "" {
				// Attempt to get title from a 'title' attribute if text is empty
				titleAttr, titleAttrExists := link.Attr("title")
				if titleAttrExists {
					topicTitle = strings.TrimSpace(strings.TrimPrefix(titleAttr, "Topic:")) // Example for "Topic: Actual Title"
				}
				if topicTitle == "" {
					return // Still no title, skip
				}
			}

			topicAbsURL, err := parsedPageURL.Parse(href)
			if err != nil {
				log.Printf("[WARNING] ExtractTopicsFromHTMLInUtil: Error parsing topic URL '%s' on page %s: %v. Skipping topic.", href, pageURL, err)
				return
			}

			u, err := url.Parse(topicAbsURL.String())
			if err != nil {
				log.Printf("[WARNING] ExtractTopicsFromHTMLInUtil: Error parsing absolute topic URL '%s' on page %s: %v. Skipping topic.", topicAbsURL.String(), pageURL, err)
				return
			}

			// Try to get topicID from query param "t", "topic", or "p" (post ID, sometimes links to post in topic)
			topicID := u.Query().Get("t")
			if topicID == "" {
				topicID = u.Query().Get("topic")
			}
			if topicID == "" {
				// If it's a link to a post, the post ID might be "p"
				postID := u.Query().Get("p")
				if postID != "" {
					// This is a simplification; real mapping from postID to topicID might be complex
					// or require another fetch. For now, use "p" + postID as a proxy if "t" is missing.
					// Or, one might decide to skip topics only identified by post ID if a direct topic ID is required.
					// topicID = "p" + postID
					// For JIT, we need the actual topic ID that matches the one in the index.
					// If "t" or "topic" query param is missing, we likely cannot reliably get the topic ID.
					log.Printf("[WARNING] ExtractTopicsFromHTMLInUtil: Topic ID (t or topic param) not found for URL '%s' with title '%s' on page %s. Skipping topic.", topicAbsURL.String(), topicTitle, pageURL)
					return
				} else {
					log.Printf("[WARNING] ExtractTopicsFromHTMLInUtil: Topic ID (t or topic param) not found for URL '%s' with title '%s' on page %s. Skipping topic.", topicAbsURL.String(), topicTitle, pageURL)
					return
				}
			}

			// Basic data.Topic population. Other fields (Replies, Views, etc.) are not available
			// from this basic extraction and would remain zero/empty. JIT is primarily for discovering *new* topic IDs.
			topicData := data.Topic{
				ID:         topicID,
				SubForumID: subForumID,
				Title:      topicTitle,
				URL:        topicAbsURL.String(),
				// Other fields like AuthorUsername, Replies, Views, LastPostUsername, LastPostTimestampRaw, IsSticky, IsLocked
				// are not typically available on the sub-forum topic listing page directly for each topic in a simple link.
				// These would be filled if we fetched each topic page, but JIT refresh usually aims to be lightweight.
			}
			topics = append(topics, topicData)
		})
	})

	if len(topics) == 0 {
		log.Printf("[DEBUG] ExtractTopicsFromHTMLInUtil: No topics extracted from page %s using selectors. This might be an empty page, selector mismatch, or all topics lacked valid IDs.", pageURL)
	}
	return topics, nil
}
