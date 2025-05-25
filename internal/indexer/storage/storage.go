package storage

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"project-waypoint/internal/indexer/logger"
	"project-waypoint/internal/indexer/topic"
)

// extractSubForumID attempts to get a forum ID from the URL query string.
func extractSubForumID(pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL '%s': %w", pageURL, err)
	}
	forumID := u.Query().Get("forum")
	if forumID != "" {
		return forumID, nil
	}
	logger.Warnf("Could not find 'forum' query parameter in URL: %s. Defaulting to 'unknown_forum' for filename.", pageURL)
	return "unknown_forum", nil
}

// SaveTopicIndex saves the collected topics to a JSON file in the specified output directory.
// The filename will be topic_index_{subForumID}.json.
func SaveTopicIndex(outputDir string, topics map[string]topic.TopicInfo, subForumBaseURL string) error {
	subForumID, err := extractSubForumID(subForumBaseURL)
	if err != nil {
		logger.Warnf("Could not extract subForumID from URL '%s' due to parsing error: %v. Using default 'unknown_forum'.", subForumBaseURL, err)
		subForumID = "unknown_forum"
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory '%s': %w", outputDir, err)
	}

	fileName := fmt.Sprintf("topic_index_%s.json", subForumID)
	filePath := filepath.Join(outputDir, fileName)

	var sortedTopics []topic.TopicInfo
	for _, t := range topics {
		sortedTopics = append(sortedTopics, t)
	}
	sort.Slice(sortedTopics, func(i, j int) bool {
		return sortedTopics[i].ID < sortedTopics[j].ID
	})

	logger.Infof("Saving %d topics to %s", len(sortedTopics), filePath)
	fileData, err := json.MarshalIndent(sortedTopics, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal topics to JSON: %w", err)
	}

	if err := os.WriteFile(filePath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write topic index to file '%s': %w", filePath, err)
	}

	logger.Infof("Successfully saved topic index to %s", filePath)
	return nil
}
