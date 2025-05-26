package indexerlogic

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"

	"waypoint_archive_scripts/pkg/data" // Assuming waypoint_archive_scripts is the module name
)

// ReadTopicIndexCSV reads a single topic index CSV file and returns a slice of Topic structs.
// The CSV is expected to have columns: TopicID, Title, URL (and SubForumID needs to be passed in).
func ReadTopicIndexCSV(filePath string, subForumID string) ([]data.Topic, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] ReadTopicIndexCSV: failed to open file %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadTopicIndexCSV: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Assuming header row, so we read and discard it.
	// If no header, comment out the next line.
	_, err = reader.Read() // Read header
	if err != nil {
		if err == io.EOF {
			log.Printf("[INFO] ReadTopicIndexCSV: empty file (or only header) %s", filePath)
			return []data.Topic{}, nil // Empty file is not an error, just no topics
		}
		log.Printf("[ERROR] ReadTopicIndexCSV: failed to read header from %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadTopicIndexCSV: failed to read header from %s: %w", filePath, err)
	}

	var topics []data.Topic = make([]data.Topic, 0) // Initialize to empty slice
	lineNumber := 1                                 // For error reporting
	for {
		lineNumber++
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("[ERROR] ReadTopicIndexCSV: error reading record at line %d from %s: %v", lineNumber, filePath, err)
			return nil, fmt.Errorf("ReadTopicIndexCSV: error reading record at line %d from %s: %w", lineNumber, filePath, err)
		}

		if len(record) < 3 {
			log.Printf("[ERROR] ReadTopicIndexCSV: invalid record at line %d in %s - expected at least 3 columns, got %d", lineNumber, filePath, len(record))
			return nil, fmt.Errorf("ReadTopicIndexCSV: invalid record at line %d in %s - expected at least 3 columns, got %d", lineNumber, filePath, len(record))
		}

		topic := data.Topic{
			ID:         record[0],
			SubForumID: subForumID,
			Title:      record[1],
			URL:        record[2],
		}
		topics = append(topics, topic)
	}
	log.Printf("[INFO] ReadTopicIndexCSV: successfully read %d topics from %s", len(topics), filePath)
	return topics, nil
}

// ReadSubForumListCSV reads the subforum list CSV file.
// Assumes CSV columns: SubForumID, SubForumName.
func ReadSubForumListCSV(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] ReadSubForumListCSV: failed to open file %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadSubForumListCSV: failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	// Assuming header row, so we read and discard it.
	_, err = reader.Read() // Read header
	if err != nil {
		if err == io.EOF {
			log.Printf("[INFO] ReadSubForumListCSV: empty file (or only header) %s", filePath)
			return make(map[string]string), nil // Empty file is not an error
		}
		log.Printf("[ERROR] ReadSubForumListCSV: failed to read header from %s: %v", filePath, err)
		return nil, fmt.Errorf("ReadSubForumListCSV: failed to read header from %s: %w", filePath, err)
	}

	subForums := make(map[string]string)
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

		if len(record) < 2 {
			log.Printf("[ERROR] ReadSubForumListCSV: invalid record at line %d in %s - expected at least 2 columns, got %d", lineNumber, filePath, len(record))
			return nil, fmt.Errorf("ReadSubForumListCSV: invalid record at line %d in %s - expected at least 2 columns, got %d", lineNumber, filePath, len(record))
		}

		subForums[record[0]] = record[1] // Map SubForumID to SubForumName
	}
	log.Printf("[INFO] ReadSubForumListCSV: successfully read %d subforum entries from %s", len(subForums), filePath)
	return subForums, nil
}
