package indexerlogic

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"waypoint_archive_scripts/pkg/data" // Assuming waypoint_archive_scripts is the module name
)

// ReadTopicIndexCSV reads a single topic index CSV file and returns a slice of Topic structs.
// Expected CSV columns: TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked
func ReadTopicIndexCSV(filePath string, subForumID string) ([]data.Topic, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] ReadTopicIndexCSV: failed to open file %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadTopicIndexCSV: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	_, err = reader.Read() // Read and discard header row
	if err != nil {
		if err == io.EOF { // Empty file or only header
			log.Printf("[INFO] ReadTopicIndexCSV: empty file (or only header) %s", filePath)
			return []data.Topic{}, nil
		}
		log.Printf("[ERROR] ReadTopicIndexCSV: failed to read header from %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadTopicIndexCSV: failed to read header from %s: %w", filePath, err)
	}

	var topics []data.Topic = make([]data.Topic, 0)
	lineNumber := 1 // Start counting data rows from 1 (after header)
	for {
		lineNumber++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			// For persistent errors like unclosed quotes, this will be the main error returned
			log.Printf("[ERROR] ReadTopicIndexCSV: error reading record at line %d from %s: %v", lineNumber, filePath, err)
			return nil, fmt.Errorf("ReadTopicIndexCSV: error reading record at line %d from %s: %w", lineNumber, filePath, err)
		}

		const expectedColumns = 10 // TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked
		if len(record) < expectedColumns {
			log.Printf("[WARNING] ReadTopicIndexCSV: skipping invalid record at line %d in %s - expected at least %d columns, got %d", lineNumber, filePath, expectedColumns, len(record))
			continue // Skip malformed row
		}

		replies, err := strconv.Atoi(record[4])
		if err != nil {
			log.Printf("[WARNING] ReadTopicIndexCSV: skipping record at line %d in %s - invalid Replies format '%s': %v", lineNumber, filePath, record[4], err)
			continue
		}
		views, err := strconv.Atoi(record[5])
		if err != nil {
			log.Printf("[WARNING] ReadTopicIndexCSV: skipping record at line %d in %s - invalid Views format '%s': %v", lineNumber, filePath, record[5], err)
			continue
		}
		isSticky, err := strconv.ParseBool(record[8])
		if err != nil {
			log.Printf("[WARNING] ReadTopicIndexCSV: skipping record at line %d in %s - invalid IsSticky format '%s': %v", lineNumber, filePath, record[8], err)
			continue
		}
		isLocked, err := strconv.ParseBool(record[9])
		if err != nil {
			log.Printf("[WARNING] ReadTopicIndexCSV: skipping record at line %d in %s - invalid IsLocked format '%s': %v", lineNumber, filePath, record[9], err)
			continue
		}

		topic := data.Topic{
			ID:                   record[0],
			SubForumID:           subForumID,
			Title:                record[1],
			URL:                  record[2],
			AuthorUsername:       record[3],
			Replies:              replies,
			Views:                views,
			LastPostUsername:     record[6],
			LastPostTimestampRaw: record[7],
			IsSticky:             isSticky,
			IsLocked:             isLocked,
		}
		topics = append(topics, topic)
	}
	log.Printf("[INFO] ReadTopicIndexCSV: successfully read %d topics from %s", len(topics), filePath)
	return topics, nil
}

// SubForumNameAndURL is a helper struct to hold name and URL read from the subforum list CSV.
// This is used as the value in the map returned by ReadSubForumListCSV.
type SubForumNameAndURL struct {
	Name string
	URL  string
}

// ReadSubForumListCSV reads the subforum list CSV file.
// Assumes CSV columns: SubForumID, SubForumName, SubForumURL.
// Returns a map of SubForumID to SubForumNameAndURL.
func ReadSubForumListCSV(filePath string) (map[string]SubForumNameAndURL, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] ReadSubForumListCSV: failed to open file %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadSubForumListCSV: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Assuming header row, so we read and discard it.
	_, err = reader.Read() // Read header: SubForumID,SubForumName,SubForumURL
	if err != nil {
		if err == io.EOF {
			log.Printf("[INFO] ReadSubForumListCSV: empty file (or only header) %s", filePath)
			return make(map[string]SubForumNameAndURL), nil // Empty file is not an error
		}
		log.Printf("[ERROR] ReadSubForumListCSV: failed to read header from %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadSubForumListCSV: failed to read header from %s: %w", filePath, err)
	}

	subForumsMap := make(map[string]SubForumNameAndURL)
	lineNumber := 1 // For error reporting
	for {
		lineNumber++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[ERROR] ReadSubForumListCSV: error reading record at line %d from %s: %v", lineNumber, filePath, err)
			return nil, fmt.Errorf("ReadSubForumListCSV: error reading record at line %d from %s: %w", lineNumber, filePath, err)
		}

		const expectedSFColumns = 3 // SubForumID, SubForumName, SubForumURL
		if len(record) < expectedSFColumns {
			log.Printf("[WARNING] ReadSubForumListCSV: invalid record at line %d in %s - expected at least %d columns, got %d. Skipping record.", lineNumber, filePath, expectedSFColumns, len(record))
			continue // Skip malformed row
		}

		subForumsMap[record[0]] = SubForumNameAndURL{
			Name: record[1],
			URL:  record[2],
		}
	}
	log.Printf("[INFO] ReadSubForumListCSV: successfully read %d subforum entries from %s", len(subForumsMap), filePath)
	return subForumsMap, nil
}
