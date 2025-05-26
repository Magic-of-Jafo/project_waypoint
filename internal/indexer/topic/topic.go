package topic

import (
	"fmt"
	// "log" // Replaced by custom logger
	"net/url"
	"strings"

	"project-waypoint/internal/indexer/logger" // Corrected to project-waypoint module

	"github.com/PuerkitoBio/goquery"
)

// TopicInfo holds the extracted information for a single forum topic.
type TopicInfo struct {
	ID    string
	Title string
	URL   string
}

// ExtractTopics parses the HTML content of a sub-forum page and extracts information
// for each topic listed.
func ExtractTopics(htmlContent string, pageURL string) ([]TopicInfo, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create goquery document: %w", err)
	}

	parsedPageURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page URL %s: %w", pageURL, err)
	}

	var topics []TopicInfo = make([]TopicInfo, 0)
	seenTopicIDs := make(map[string]bool)

	// Topics are in <tr> elements. The relevant <a> tag is usually within the second <td>.
	// We'll look for <tr> elements that directly contain <td> elements,
	// then inspect the <a> tags within those, specifically those linking to 'viewtopic.php'.
	doc.Find("table.normal tr").Each(func(i int, tr *goquery.Selection) {
		// Find <a> tags that are direct children of <td> with class 'normal bgc2' and link to viewtopic.php
		tr.Find("td.normal.bgc2 > a.b[href*='viewtopic.php']").Each(func(j int, link *goquery.Selection) {
			href, exists := link.Attr("href")
			if !exists {
				// Log or handle missing href if necessary, though selector should ensure it
				return
			}

			topicTitle := strings.TrimSpace(link.Text())
			if topicTitle == "" {
				// Skip if title is empty
				return
			}

			// Resolve the topic URL relative to the page's URL
			topicAbsURL, err := parsedPageURL.Parse(href)
			if err != nil {
				logger.Warnf("Error parsing topic URL '%s': %v. Skipping topic.", href, err)
				return
			}

			u, err := url.Parse(topicAbsURL.String())
			if err != nil {
				logger.Warnf("Error parsing absolute topic URL '%s': %v. Skipping topic.", topicAbsURL.String(), err)
				return
			}
			topicID := u.Query().Get("topic")

			if topicID == "" {
				logger.Warnf("Topic ID not found for URL '%s' with title '%s'. Skipping topic.", topicAbsURL.String(), topicTitle)
				return
			}

			// AC9: Ensure de-duplication of results from a single page.
			if !seenTopicIDs[topicID] {
				topics = append(topics, TopicInfo{
					ID:    topicID,
					Title: topicTitle,
					URL:   topicAbsURL.String(),
				})
				seenTopicIDs[topicID] = true
			}
		})
	})

	if len(topics) == 0 {
		// If doc creation was fine but no topics found, it might be a selector issue or empty page.
		// Check if the doc.Find had any matches at all to differentiate.
		// For now, a debug or info message is fine.
		logger.Debugf("No topics extracted from page: %s. This might be an empty page or selector mismatch.", pageURL)
	}

	return topics, nil
}

// TODO: Implement topic ID & URL extraction logic here
