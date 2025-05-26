package indexerlogic

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/data" // Assuming waypoint_archive_scripts is the module name
)

// ProcessTopicsAndSubForums counts topics per sub-forum and prepares SubForum structs.
// allTopics would be a slice of all Topic structs read from all index files.
// subForumDetails would be map[string]SubForumNameAndURL from ReadSubForumListCSV.
func ProcessTopicsAndSubForums(allTopics []data.Topic, subForumDetails map[string]SubForumNameAndURL) ([]data.SubForum, error) {
	topicsBySubForum := make(map[string][]data.Topic)
	for _, topic := range allTopics {
		topicsBySubForum[topic.SubForumID] = append(topicsBySubForum[topic.SubForumID], topic)
	}

	var subForums []data.SubForum
	for sfID, sfTopics := range topicsBySubForum {
		details, ok := subForumDetails[sfID]
		if !ok {
			log.Printf("[WARNING] ProcessTopicsAndSubForums: SubForumID '%s' found in topic data but not in subforum list. Using ID as name and empty URL.", sfID)
			details = SubForumNameAndURL{Name: sfID, URL: ""} // Use ID as name, empty URL if not found
		}
		subForum := data.SubForum{
			ID:         sfID,
			Name:       details.Name,
			URL:        details.URL, // Populate the URL
			TopicCount: len(sfTopics),
			Topics:     sfTopics,
		}
		subForums = append(subForums, subForum)
	}
	return subForums, nil
}

// SortSubForumsByTopicCount sorts a slice of SubForum structs by their TopicCount in ascending order.
func SortSubForumsByTopicCount(subForums []data.SubForum) {
	sort.Slice(subForums, func(i, j int) bool {
		return subForums[i].TopicCount < subForums[j].TopicCount
	})
}

// GenerateMasterTopicList creates a single list of all topics, ordered by sorted sub-forums,
// and ensures global de-duplication of Topic IDs.
func GenerateMasterTopicList(sortedSubForums []data.SubForum) data.MasterTopicList {
	masterList := data.MasterTopicList{Topics: []data.Topic{}}
	seenTopicIDs := make(map[string]bool)

	for _, sf := range sortedSubForums {
		// To ensure topics within a sub-forum are also consistently ordered (e.g., by ID or original order if preserved)
		// you might want to sort sf.Topics here if that's a requirement.
		// For now, we assume the order from the CSV is sufficient or doesn't matter within a sub-forum batch.
		for _, topic := range sf.Topics {
			if !seenTopicIDs[topic.ID] {
				masterList.Topics = append(masterList.Topics, topic)
				seenTopicIDs[topic.ID] = true
			}
		}
	}
	return masterList
}

// LoadAndProcessTopicIndex orchestrates the loading, processing, and sorting of topic index data.
// It logs information as per AC8.
// It now returns the sorted slice of SubForum structs along with the MasterTopicList.
func LoadAndProcessTopicIndex(cfg *config.Config) ([]data.SubForum, data.MasterTopicList, error) {
	log.Println("[INFO] Starting Topic Index Consumption and Prioritization Logic...")

	subForumDetails, err := ReadSubForumListCSV(cfg.SubForumListFile)
	if err != nil {
		log.Printf("[FATAL] Failed to load subforum list from %s: %v", cfg.SubForumListFile, err)
		return nil, data.MasterTopicList{}, fmt.Errorf("LoadAndProcessTopicIndex: failed to load subforum list: %w", err)
	}
	log.Printf("[INFO] Successfully loaded %d entries from subforum list: %s", len(subForumDetails), cfg.SubForumListFile)

	var allTopics []data.Topic
	files, err := os.ReadDir(cfg.TopicIndexDir)
	if err != nil {
		log.Printf("[FATAL] Failed to read topic index directory %s: %v", cfg.TopicIndexDir, err)
		return nil, data.MasterTopicList{}, fmt.Errorf("LoadAndProcessTopicIndex: failed to read topic index directory: %w", err)
	}

	log.Printf("[INFO] Reading topic index files from directory: %s", cfg.TopicIndexDir)
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "topic_index_") && strings.HasSuffix(file.Name(), ".csv") {
			filePath := filepath.Join(cfg.TopicIndexDir, file.Name())
			// Extract SubForumID from filename, e.g., topic_index_123.csv -> 123
			nameParts := strings.Split(strings.TrimSuffix(file.Name(), ".csv"), "_")
			if len(nameParts) < 3 { // topic_index_ID
				log.Printf("[WARNING] Skipping file with unexpected name format: %s", file.Name())
				continue
			}
			subForumID := nameParts[len(nameParts)-1]

			topics, err := ReadTopicIndexCSV(filePath, subForumID)
			if err != nil {
				log.Printf("[WARNING] Failed to read or parse topic index file %s: %v. Skipping this file.", filePath, err)
				// Decide if one failed file should halt everything or just be skipped.
				// For now, skipping as per AC9 suggests halting for unparsable *format* globally, not one bad file.
				continue
			}
			allTopics = append(allTopics, topics...)
		}
	}
	log.Printf("[INFO] Finished reading all topic index files. Total raw topic entries: %d", len(allTopics))

	subForums, err := ProcessTopicsAndSubForums(allTopics, subForumDetails)
	if err != nil {
		// This function doesn't currently return an error, but good practice for the future.
		log.Printf("[FATAL] Failed to process topics and subforums: %v", err)
		return nil, data.MasterTopicList{}, fmt.Errorf("LoadAndProcessTopicIndex: failed to process topics: %w", err)
	}

	SortSubForumsByTopicCount(subForums)
	log.Println("[INFO] Sub-forums sorted by topic count (ascending). Determined processing order:")
	for _, sf := range subForums {
		log.Printf("[INFO] - Sub-forum: %s (ID: %s, Topics: %d)", sf.Name, sf.ID, sf.TopicCount)
	}

	masterTopicList := GenerateMasterTopicList(subForums)
	log.Printf("[INFO] Master topic list generated. Total unique Topic IDs successfully loaded: %d", len(masterTopicList.Topics))

	log.Println("[INFO] Topic Index Consumption and Prioritization Logic completed.")
	return subForums, masterTopicList, nil
}
