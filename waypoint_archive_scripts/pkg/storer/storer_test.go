package storer

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSaveTopicHTML_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_archive_success")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	archiveRoot := tempDir
	subForumID := "sf123"
	topicID := "topic456"
	pageNumber := 1
	htmlContent := []byte("<html><body>Test Content</body></html>")

	expectedFileName := "page_1.html"
	expectedTopicPath := filepath.Join(archiveRoot, subForumID, topicID)
	expectedFullPath := filepath.Join(expectedTopicPath, expectedFileName)

	savedPath, err := SaveTopicHTML(archiveRoot, subForumID, topicID, pageNumber, htmlContent)
	if err != nil {
		t.Fatalf("SaveTopicHTML failed: %v", err)
	}

	if savedPath != expectedFullPath {
		t.Errorf("Expected saved path %s, got %s", expectedFullPath, savedPath)
	}

	// Verify directory creation
	if _, err := os.Stat(expectedTopicPath); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to be created, but it was not", expectedTopicPath)
	}

	// Verify file creation and content
	content, err := os.ReadFile(expectedFullPath)
	if err != nil {
		t.Fatalf("Failed to read saved file %s: %v", expectedFullPath, err)
	}
	if !bytes.Equal(content, htmlContent) {
		t.Errorf("Expected file content %s, got %s", string(htmlContent), string(content))
	}
}

func TestSaveTopicHTML_Overwrite(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_archive_overwrite")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	archiveRoot := tempDir
	subForumID := "sf_overwrite"
	topicID := "topic_overwrite"
	pageNumber := 1
	initialContent := []byte("<html><body>Initial Content</body></html>")
	newContent := []byte("<html><body>New Overwritten Content</body></html>")

	expectedFileName := "page_1.html"
	expectedFullPath := filepath.Join(archiveRoot, subForumID, topicID, expectedFileName)

	// Save initial file
	_, err = SaveTopicHTML(archiveRoot, subForumID, topicID, pageNumber, initialContent)
	if err != nil {
		t.Fatalf("SaveTopicHTML (initial save) failed: %v", err)
	}

	// Save new file to overwrite
	_, err = SaveTopicHTML(archiveRoot, subForumID, topicID, pageNumber, newContent)
	if err != nil {
		t.Fatalf("SaveTopicHTML (overwrite save) failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(expectedFullPath)
	if err != nil {
		t.Fatalf("Failed to read saved file %s: %v", expectedFullPath, err)
	}
	if !bytes.Equal(content, newContent) {
		t.Errorf("Expected file content to be overwritten with %s, got %s", string(newContent), string(content))
	}
}

func TestSaveTopicHTML_PathCleaning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_archive_path_cleaning")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	archiveRoot := tempDir
	// Intentionally include path elements that should be cleaned
	subForumID := "sf//.././user_input"
	topicID := "topic/./bad/../example"
	pageNumber := 1
	htmlContent := []byte("<html><body>Clean Path Test</body></html>")

	// filepath.Clean behavior:
	// "sf//.././user_input" -> "user_input" (assuming archiveRoot is not just "/")
	// "topic/./bad/../example" -> "topic/example"

	// Note: if archiveRoot was "/", "sf//.././user_input" might become "../user_input"
	// However, MkdirTemp creates a deep path, so ".." will resolve within tempDir.
	// For "sf//.././user_input", if archiveRoot = /tmp/test123
	// then path is /tmp/test123/sf/../../user_input -> /tmp/user_input which is outside our archive root.
	// filepath.Join(archiveRoot, filepath.Clean(subForumID)) is safer.
	// The current implementation does: topicPath := filepath.Join(archiveRoot, safeSubForumID, safeTopicID)
	// where safeSubForumID = filepath.Clean(subForumID)
	// This means filepath.Join(tempDir, filepath.Clean("sf//.././user_input"))
	// filepath.Clean("sf//.././user_input") = "sf/../user_input"
	// filepath.Join(tempDir, "sf/../user_input") = tempDir/user_input (if tempDir doesn't end with /)
	// This needs careful checking. For now, let's test the *expected output* of current code.

	// Current implementation:
	// safeSubForumID := filepath.Clean("sf//.././user_input") -> "sf/../user_input"
	// safeTopicID := filepath.Clean("topic/./bad/../example") -> "topic/example"
	// topicPath := filepath.Join(tempDir, "sf/../user_input", "topic/example") -> tempDir/user_input/topic/example

	expectedSubForumComponent := "user_input" // Based on how filepath.Join(tempDir, "sf/../user_input") works
	expectedTopicComponent := "topic/example"

	expectedFileName := "page_1.html"
	// The actual created path will be: tempDir/user_input/topic/example/page_1.html
	expectedFullPath := filepath.Join(archiveRoot, expectedSubForumComponent, expectedTopicComponent, expectedFileName)

	savedPath, err := SaveTopicHTML(archiveRoot, subForumID, topicID, pageNumber, htmlContent)
	if err != nil {
		t.Fatalf("SaveTopicHTML with dirty paths failed: %v", err)
	}

	if savedPath != expectedFullPath {
		t.Errorf("Expected cleaned saved path %s, got %s", expectedFullPath, savedPath)
	}

	if _, err := os.Stat(expectedFullPath); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created with cleaned path, but it was not", expectedFullPath)
	}

	// Check that we didn't create a "sf" directory directly under tempDir
	// or a "topic/bad" directory.
	unwantedPath1 := filepath.Join(archiveRoot, "sf")
	if _, err := os.Stat(unwantedPath1); !os.IsNotExist(err) {
		t.Errorf("Unwanted directory component %s was created", unwantedPath1)
	}
}

func TestSaveTopicHTML_Error_MkdirAllFails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_archive_mkdir_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a file where a directory is expected, to make MkdirAll fail
	archiveRoot := filepath.Join(tempDir, "existing_file.txt")
	if err := os.WriteFile(archiveRoot, []byte("I am a file"), 0644); err != nil {
		t.Fatalf("Failed to create conflicting file: %v", err)
	}

	subForumID := "sf_error"
	topicID := "topic_error"
	pageNumber := 1
	htmlContent := []byte("<html><body>Error Test</body></html>")

	_, err = SaveTopicHTML(archiveRoot, subForumID, topicID, pageNumber, htmlContent)
	if err == nil {
		t.Fatal("Expected an error when MkdirAll fails, but got nil")
		return
	}

	// Check if the error message contains the expected information
	expectedErrorMsgPart := "Error creating directory"
	if !strings.Contains(err.Error(), expectedErrorMsgPart) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsgPart, err.Error())
	}
}

func TestSaveTopicHTML_Error_WriteFileFails(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test_archive_writefile_error")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	archiveRoot := tempDir
	subForumID := "sf_write_error"
	topicID := "topic_write_error"
	pageNumber := 1
	htmlContent := []byte("<html><body>Write Error Test</body></html>")

	// Create a directory where the file is supposed to be written, to make WriteFile fail
	expectedFileName := "page_1.html"
	conflictPath := filepath.Join(archiveRoot, subForumID, topicID, expectedFileName)
	if err := os.MkdirAll(conflictPath, 0755); err != nil { // Create a dir with the target filename
		t.Fatalf("Failed to create conflicting directory: %v", err)
	}

	_, err = SaveTopicHTML(archiveRoot, subForumID, topicID, pageNumber, htmlContent)
	if err == nil {
		t.Fatal("Expected an error when WriteFile fails, but got nil")
		return
	}

	expectedErrorMsgPart := "Error writing file"
	if !strings.Contains(err.Error(), expectedErrorMsgPart) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedErrorMsgPart, err.Error())
	}
}
