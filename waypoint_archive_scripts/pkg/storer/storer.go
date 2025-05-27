package storer

import (
	"fmt"
	"os"
	"path/filepath"
)

// Storer handles the saving of HTML content to the file system.
// It ensures that the directory structure <ArchiveOutputRootDir>/<SubForumID>/<TopicID>/page_<PageNum>.html is used.
type Storer struct {
	ArchiveOutputRootDir string
}

// NewStorer creates a new Storer instance.
// archiveOutputRootDir is the base directory where all archived content will be saved.
func NewStorer(archiveOutputRootDir string) *Storer {
	return &Storer{
		ArchiveOutputRootDir: archiveOutputRootDir,
	}
}

// SaveTopicHTML saves the HTML content of a specific topic page.
// It creates the necessary directory structure if it doesn't exist.
// Returns the full path to the saved file or an error.
func (s *Storer) SaveTopicHTML(subForumID, topicID string, pageNum int, htmlBytes []byte) (string, error) {
	topicDir := filepath.Join(s.ArchiveOutputRootDir, subForumID, topicID)
	err := os.MkdirAll(topicDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create topic directory %s: %w", topicDir, err)
	}

	fileName := fmt.Sprintf("page_%d.html", pageNum)
	filePath := filepath.Join(topicDir, fileName)

	err = os.WriteFile(filePath, htmlBytes, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write HTML file %s: %w", filePath, err)
	}

	// log.Printf("[DEBUG] Storer: Successfully saved HTML file: %s (Size: %d bytes)", filePath, len(htmlBytes))
	return filePath, nil
}
