package storer

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// SaveTopicHTML saves the raw HTML content for a specific topic page to disk.
// It constructs the path according to: {ARCHIVE_ROOT}/{sub_forum_id_or_name}/{topic_id}/page_{page_number}.html
// It will create necessary directories if they don't exist.
// It will overwrite the file if it already exists.
func SaveTopicHTML(
	archiveRoot string,
	subForumID string, // Can be name or ID
	topicID string,
	pageNumber int,
	htmlContent []byte,
) (string, error) {
	// Task 2: Implement Directory Structure Management (AC: 2, 3, 4, 9)
	// Subtask 2.2: Construct the full file path
	fileName := fmt.Sprintf("page_%d.html", pageNumber)
	// Sanitize subForumID and topicID to ensure they are valid path components
	// For now, we assume they are valid, but this is a place for future enhancement if needed.
	safeSubForumID := filepath.Clean(subForumID) // Basic cleaning
	safeTopicID := filepath.Clean(topicID)       // Basic cleaning

	topicPath := filepath.Join(archiveRoot, safeSubForumID, safeTopicID)
	fullPath := filepath.Join(topicPath, fileName)

	// Subtask 2.3: Implement logic to create sub-directories
	// 0755 permissions: rwxr-xr-x
	err := os.MkdirAll(topicPath, 0755)
	if err != nil {
		errMsg := fmt.Sprintf("Error creating directory %s: %v", topicPath, err)
		log.Println(errMsg) // Subtask 4.2: Log errors clearly
		return "", fmt.Errorf(errMsg)
	}

	// Task 1: Implement File Saving Functionality (AC: 1, 8)
	// Subtask 1.2: Implement logic to write the provided HTML content to a file.
	// Task 3: Implement File Overwriting (AC: 5) - os.WriteFile handles overwriting by default.
	// 0644 permissions: rw-r--r--
	err = os.WriteFile(fullPath, htmlContent, 0644)
	if err != nil {
		errMsg := fmt.Sprintf("Error writing file %s: %v", fullPath, err)
		log.Println(errMsg) // Subtask 4.2: Log errors clearly
		return "", fmt.Errorf(errMsg)
	}

	// Subtask 4.3: Log successful file save operations, including path and file size.
	fileInfo, err := os.Stat(fullPath)
	fileSize := int64(-1) // Default to -1 if stat fails
	if err == nil {
		fileSize = fileInfo.Size()
	}
	log.Printf("Successfully saved HTML file: %s (Size: %d bytes)", fullPath, fileSize)

	return fullPath, nil
}
