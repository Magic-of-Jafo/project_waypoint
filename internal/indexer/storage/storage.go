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

// ExtractSubForumID attempts to get a forum ID from the URL query string.
// It is an exported function.
func ExtractSubForumID(pageURL string) (string, error) {
	u, err := url.Parse(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL '%s': %w", pageURL, err)
	}
	forumID := u.Query().Get("forum")
	if forumID != "" {
		return forumID, nil
	}
	// If 'forum' parameter is not found, it's not necessarily an error for this function's purpose,
	// but the caller might treat an empty string as such. For robustness, let's return an empty string
	// and let the caller decide on "unknown_forum" or other fallbacks.
	// logger.Warnf within this specific utility might be too chatty if used by various parts.
	return "", nil // Return empty string if not found, let caller handle fallback if needed.
}

// SaveTopicIndex saves the collected topics to a JSON file in the specified output directory.
// The filename will be topic_index_{subForumID}.json.
func SaveTopicIndex(outputDir string, topics map[string]topic.TopicInfo, subForumBaseURL string) error {
	subForumID, err := ExtractSubForumID(subForumBaseURL)
	if err != nil {
		logger.Warnf("Could not extract subForumID from URL '%s' due to parsing error: %v. Using default 'unknown_forum' for filename.", subForumBaseURL, err)
		subForumID = "unknown_forum"
	} else if subForumID == "" {
		logger.Warnf("'forum' query parameter not found in URL '%s'. Using default 'unknown_forum' for filename.", subForumBaseURL)
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
