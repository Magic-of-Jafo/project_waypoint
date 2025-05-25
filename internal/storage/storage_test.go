package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInitializeStorage(t *testing.T) {
	basePath, err := os.MkdirTemp("", "storage_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath) // Clean up

	// First call to InitializeStorage
	err = InitializeStorage(basePath)
	if err != nil {
		t.Fatalf("InitializeStorage failed: %v", err)
	}

	// Check if directories were created
	expectedDirs := []string{
		filepath.Join(basePath, "raw-html"),
		filepath.Join(basePath, "structured-json"),
		filepath.Join(basePath, "metadata"),
	}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Directory %s was not created", dir)
		}
	}

	// Check if progress.json was created and has correct initial content
	progressFilePath := filepath.Join(basePath, "progress.json")
	if _, err := os.Stat(progressFilePath); os.IsNotExist(err) {
		t.Fatalf("progress.json was not created at %s", progressFilePath)
	}

	fileContent, err := os.ReadFile(progressFilePath)
	if err != nil {
		t.Fatalf("Failed to read progress.json: %v", err)
	}

	var progressData ProgressData
	err = json.Unmarshal(fileContent, &progressData)
	if err != nil {
		t.Fatalf("Failed to unmarshal progress.json: %v", err)
	}

	expectedProgressData := ProgressData{
		OverallArchivalProgress: 0.0,
		LastProcessedSubForum:   "",
		LastProcessedTopic:      "",
		LastProcessedPage:       "",
	}

	if progressData != expectedProgressData {
		t.Errorf("progress.json content mismatch. Got %+v, expected %+v", progressData, expectedProgressData)
	}

	// Modify progress.json and call InitializeStorage again to ensure it's not overwritten
	modifiedProgressData := ProgressData{
		OverallArchivalProgress: 50.0,
		LastProcessedSubForum:   "test-forum",
		LastProcessedTopic:      "test-topic",
		LastProcessedPage:       "test-page",
	}
	jsonData, err := json.MarshalIndent(modifiedProgressData, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal modified progress data: %v", err)
	}
	err = os.WriteFile(progressFilePath, jsonData, 0644)
	if err != nil {
		t.Fatalf("Failed to write modified progress.json: %v", err)
	}

	// Second call to InitializeStorage
	err = InitializeStorage(basePath)
	if err != nil {
		t.Fatalf("InitializeStorage (second call) failed: %v", err)
	}

	// Check content of progress.json again, it should be the modified content
	fileContentAfterSecondCall, err := os.ReadFile(progressFilePath)
	if err != nil {
		t.Fatalf("Failed to read progress.json after second call: %v", err)
	}

	var progressDataAfterSecondCall ProgressData
	err = json.Unmarshal(fileContentAfterSecondCall, &progressDataAfterSecondCall)
	if err != nil {
		t.Fatalf("Failed to unmarshal progress.json after second call: %v", err)
	}

	if progressDataAfterSecondCall != modifiedProgressData {
		t.Errorf("progress.json was overwritten. Got %+v, expected %+v", progressDataAfterSecondCall, modifiedProgressData)
	}
}

func TestGetRawHTMLPath(t *testing.T) {
	expected := filepath.Join("base", "raw-html", "subforum-sf1", "topic-t1", "page-1.html")
	actual := GetRawHTMLPath("base", "sf1", "t1", "1")
	if actual != expected {
		t.Errorf("GetRawHTMLPath failed: expected %s, got %s", expected, actual)
	}
}

func TestGetStructuredJSONPath(t *testing.T) {
	expected := filepath.Join("base", "structured-json", "subforum-sf1", "topic-t1.json")
	actual := GetStructuredJSONPath("base", "sf1", "t1")
	if actual != expected {
		t.Errorf("GetStructuredJSONPath failed: expected %s, got %s", expected, actual)
	}
}

func TestGetSubForumMetadataIndexPath(t *testing.T) {
	expected := filepath.Join("base", "metadata", "subforum-sf1", "index.json")
	actual := GetSubForumMetadataIndexPath("base", "sf1")
	if actual != expected {
		t.Errorf("GetSubForumMetadataIndexPath failed: expected %s, got %s", expected, actual)
	}
}

func TestWriteSubForumMetadata(t *testing.T) {
	basePath, err := os.MkdirTemp("", "metadata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath)

	// It's good practice to initialize the base storage, though WriteSubForumMetadata should also create its own path.
	if err := InitializeStorage(basePath); err != nil {
		t.Fatalf("InitializeStorage failed: %v", err)
	}

	subForumID := "sfTest123"
	initialMetadata := SubForumMetadata{
		TotalTopics: 10,
		PagesPerTopic: map[string]int{
			"topicA": 5,
			"topicB": 3,
		},
		LastUpdateTimestamp: "2023-01-01T10:00:00Z",
	}

	err = WriteSubForumMetadata(basePath, subForumID, initialMetadata)
	if err != nil {
		t.Fatalf("WriteSubForumMetadata failed: %v", err)
	}

	metadataFilePath := GetSubForumMetadataIndexPath(basePath, subForumID)
	if _, err := os.Stat(metadataFilePath); os.IsNotExist(err) {
		t.Fatalf("Metadata file %s was not created", metadataFilePath)
	}

	fileContent, err := os.ReadFile(metadataFilePath)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	var actualMetadata SubForumMetadata
	err = json.Unmarshal(fileContent, &actualMetadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal metadata file: %v", err)
	}

	// Basic check, for more complex map comparisons, reflect.DeepEqual might be needed
	if actualMetadata.TotalTopics != initialMetadata.TotalTopics ||
		actualMetadata.LastUpdateTimestamp != initialMetadata.LastUpdateTimestamp ||
		len(actualMetadata.PagesPerTopic) != len(initialMetadata.PagesPerTopic) {
		t.Errorf("Initial metadata content mismatch. Got %+v, expected %+v", actualMetadata, initialMetadata)
	}
	for k, v := range initialMetadata.PagesPerTopic {
		if actualMetadata.PagesPerTopic[k] != v {
			t.Errorf("Initial metadata PagesPerTopic mismatch for key %s. Got %d, expected %d", k, actualMetadata.PagesPerTopic[k], v)
		}
	}

	// Test overwrite
	updatedMetadata := SubForumMetadata{
		TotalTopics:         15,
		PagesPerTopic:       map[string]int{"topicC": 7},
		LastUpdateTimestamp: "2023-01-02T12:00:00Z",
	}

	err = WriteSubForumMetadata(basePath, subForumID, updatedMetadata)
	if err != nil {
		t.Fatalf("WriteSubForumMetadata (overwrite) failed: %v", err)
	}

	fileContent, err = os.ReadFile(metadataFilePath)
	if err != nil {
		t.Fatalf("Failed to read metadata file (overwrite): %v", err)
	}

	// Re-initialize actualMetadata or its map to ensure a clean unmarshal for the overwrite check
	actualMetadata = SubForumMetadata{} // Or actualMetadata.PagesPerTopic = make(map[string]int)
	err = json.Unmarshal(fileContent, &actualMetadata)
	if err != nil {
		t.Fatalf("Failed to unmarshal metadata file (overwrite): %v", err)
	}

	if actualMetadata.TotalTopics != updatedMetadata.TotalTopics ||
		actualMetadata.LastUpdateTimestamp != updatedMetadata.LastUpdateTimestamp ||
		len(actualMetadata.PagesPerTopic) != len(updatedMetadata.PagesPerTopic) {
		t.Errorf("Updated metadata content mismatch. Got %+v, expected %+v", actualMetadata, updatedMetadata)
	}
	for k, v := range updatedMetadata.PagesPerTopic {
		if actualMetadata.PagesPerTopic[k] != v {
			t.Errorf("Updated metadata PagesPerTopic mismatch for key %s. Got %d, expected %d", k, actualMetadata.PagesPerTopic[k], v)
		}
	}
}

func TestReadWriteProgressFile(t *testing.T) {
	basePath, err := os.MkdirTemp("", "progress_rw_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath)

	// Initialize storage to create the initial progress.json
	if err := InitializeStorage(basePath); err != nil {
		t.Fatalf("InitializeStorage failed: %v", err)
	}

	// Test ReadProgressFile with initial data
	initialData, err := ReadProgressFile(basePath)
	if err != nil {
		t.Fatalf("ReadProgressFile (initial) failed: %v", err)
	}
	expectedInitialData := ProgressData{OverallArchivalProgress: 0.0, LastProcessedSubForum: "", LastProcessedTopic: "", LastProcessedPage: ""}
	if initialData != expectedInitialData {
		t.Errorf("ReadProgressFile (initial) content mismatch. Got %+v, expected %+v", initialData, expectedInitialData)
	}

	// Test WriteProgressFile
	updatedData := ProgressData{
		OverallArchivalProgress: 75.5,
		LastProcessedSubForum:   "sf100",
		LastProcessedTopic:      "t200",
		LastProcessedPage:       "p5",
	}
	err = WriteProgressFile(basePath, updatedData)
	if err != nil {
		t.Fatalf("WriteProgressFile failed: %v", err)
	}

	// Test ReadProgressFile with updated data
	dataAfterWrite, err := ReadProgressFile(basePath)
	if err != nil {
		t.Fatalf("ReadProgressFile (after write) failed: %v", err)
	}
	if dataAfterWrite != updatedData {
		t.Errorf("ReadProgressFile (after write) content mismatch. Got %+v, expected %+v", dataAfterWrite, updatedData)
	}

	// Test ReadProgressFile with invalid progress value (-1)
	invalidProgressDataNeg := ProgressData{OverallArchivalProgress: -1.0}
	if err = WriteProgressFile(basePath, invalidProgressDataNeg); err != nil {
		t.Fatalf("WriteProgressFile for invalid data (neg) failed: %v", err)
	}
	// Expect ErrStorageInvalidFormat because data.OverallArchivalProgress is out of range
	if _, err = ReadProgressFile(basePath); !errors.Is(err, ErrStorageInvalidFormat) {
		t.Errorf("ReadProgressFile wrong error for negative OverallArchivalProgress. Got %T, %v, want wrapping %T", err, err, ErrStorageInvalidFormat)
	}

	// Test ReadProgressFile with invalid progress value (101)
	invalidProgressDataPos := ProgressData{OverallArchivalProgress: 101.0}
	if err = WriteProgressFile(basePath, invalidProgressDataPos); err != nil {
		t.Fatalf("WriteProgressFile for invalid data (pos) failed: %v", err)
	}
	// Expect ErrStorageInvalidFormat
	if _, err = ReadProgressFile(basePath); !errors.Is(err, ErrStorageInvalidFormat) {
		t.Errorf("ReadProgressFile wrong error for OverallArchivalProgress > 100. Got %T, %v, want wrapping %T", err, err, ErrStorageInvalidFormat)
	}

	// Test ReadProgressFile with a non-existent file (by removing it first)
	progressFilePath := filepath.Join(basePath, "progress.json")
	if err := os.Remove(progressFilePath); err != nil {
		t.Fatalf("Failed to remove progress.json for testing non-existent read: %v", err)
	}
	// Expect ErrStorageNotFound
	_, err = ReadProgressFile(basePath)
	if !errors.Is(err, ErrStorageNotFound) {
		t.Errorf("ReadProgressFile wrong error for non-existent file. Got %T, %v, want wrapping %T", err, err, ErrStorageNotFound)
	}
}

func TestValidateBaseStorageStructure(t *testing.T) {
	basePath, err := os.MkdirTemp("", "validate_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath)

	// Case 1: Valid structure
	if err := InitializeStorage(basePath); err != nil {
		t.Fatalf("InitializeStorage failed: %v", err)
	}
	if err := ValidateBaseStorageStructure(basePath); err != nil {
		t.Errorf("ValidateBaseStorageStructure failed for a valid structure: %v", err)
	}

	// Case 2: Missing a directory (e.g., raw-html)
	rawHTMLPath := filepath.Join(basePath, "raw-html")
	if err := os.RemoveAll(rawHTMLPath); err != nil {
		t.Fatalf("Failed to remove raw-html for testing: %v", err)
	}
	if err := ValidateBaseStorageStructure(basePath); err == nil {
		t.Errorf("ValidateBaseStorageStructure should have failed when raw-html dir is missing")
	} else if !os.IsNotExist(err) {
		// Check if the error is specifically about the missing path, might need more specific error handling in Validate function to test this precisely
		t.Logf("ValidateBaseStorageStructure failed as expected when dir missing, error: %v", err) // Log for info
	}

	// Re-initialize for next test case
	if err := InitializeStorage(basePath); err != nil {
		t.Fatalf("InitializeStorage failed: %v", err)
	}

	// Case 3: Missing progress.json
	progressFilePath := filepath.Join(basePath, "progress.json")
	if err := os.Remove(progressFilePath); err != nil {
		t.Fatalf("Failed to remove progress.json for testing: %v", err)
	}
	if err := ValidateBaseStorageStructure(basePath); err == nil {
		t.Errorf("ValidateBaseStorageStructure should have failed when progress.json is missing")
	} else if !os.IsNotExist(err) {
		t.Logf("ValidateBaseStorageStructure failed as expected when progress.json missing, error: %v", err) // Log for info
	}
}

func TestParseRawHTMLPath(t *testing.T) {
	tests := []struct {
		name             string
		filePath         string
		wantBasePath     string
		wantSubForumID   string
		wantTopicID      string
		wantPageNumber   string
		wantErr          bool
		expectedErrorMsg string // Optional: check for specific error message content
	}{
		{
			name:           "valid path",
			filePath:       filepath.Join("c:", "archive", "raw-html", "subforum-123", "topic-456", "page-7.html"),
			wantBasePath:   filepath.Join("c:", "archive"),
			wantSubForumID: "123",
			wantTopicID:    "456",
			wantPageNumber: "7",
			wantErr:        false,
		},
		{
			name:           "valid path with dot in base",
			filePath:       filepath.Join(".", "data", "raw-html", "subforum-sf", "topic-t", "page-1.html"),
			wantBasePath:   filepath.Join(".", "data"),
			wantSubForumID: "sf",
			wantTopicID:    "t",
			wantPageNumber: "1",
			wantErr:        false,
		},
		{
			name:             "invalid structure - wrong root dir",
			filePath:         filepath.Join("c:", "archive", "wrong-dir", "subforum-123", "topic-456", "page-7.html"),
			wantErr:          true,
			expectedErrorMsg: "invalid raw HTML path structure",
		},
		{
			name:             "invalid structure - missing topic prefix",
			filePath:         filepath.Join("c:", "archive", "raw-html", "subforum-123", "456", "page-7.html"),
			wantErr:          true,
			expectedErrorMsg: "invalid raw HTML path structure",
		},
		{
			name:             "invalid structure - missing page suffix",
			filePath:         filepath.Join("c:", "archive", "raw-html", "subforum-123", "topic-456", "page-7"),
			wantErr:          true,
			expectedErrorMsg: "invalid raw HTML path structure",
		},
		{
			name:             "empty subforum ID",
			filePath:         filepath.Join("c:", "archive", "raw-html", "subforum-", "topic-456", "page-7.html"),
			wantErr:          true,
			expectedErrorMsg: "empty ID component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBasePath, gotSubForumID, gotTopicID, gotPageNumber, err := ParseRawHTMLPath(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRawHTMLPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.expectedErrorMsg != "" && (err == nil || !strings.Contains(err.Error(), tt.expectedErrorMsg)) {
					t.Errorf("ParseRawHTMLPath() error = %v, expected to contain %s", err, tt.expectedErrorMsg)
				}
				return // No further checks if error was expected
			}
			if gotBasePath != tt.wantBasePath {
				t.Errorf("ParseRawHTMLPath() gotBasePath = %v, want %v", gotBasePath, tt.wantBasePath)
			}
			if gotSubForumID != tt.wantSubForumID {
				t.Errorf("ParseRawHTMLPath() gotSubForumID = %v, want %v", gotSubForumID, tt.wantSubForumID)
			}
			if gotTopicID != tt.wantTopicID {
				t.Errorf("ParseRawHTMLPath() gotTopicID = %v, want %v", gotTopicID, tt.wantTopicID)
			}
			if gotPageNumber != tt.wantPageNumber {
				t.Errorf("ParseRawHTMLPath() gotPageNumber = %v, want %v", gotPageNumber, tt.wantPageNumber)
			}
		})
	}
}

func TestParseStructuredJSONPath(t *testing.T) {
	tests := []struct {
		name             string
		filePath         string
		wantBasePath     string
		wantSubForumID   string
		wantTopicID      string
		wantErr          bool
		expectedErrorMsg string
	}{
		{
			name:           "valid path",
			filePath:       filepath.Join("my", "data", "structured-json", "subforum-abc", "topic-def.json"),
			wantBasePath:   filepath.Join("my", "data"),
			wantSubForumID: "abc",
			wantTopicID:    "def",
			wantErr:        false,
		},
		{
			name:             "invalid structure - wrong extension",
			filePath:         filepath.Join("my", "data", "structured-json", "subforum-abc", "topic-def.txt"),
			wantErr:          true,
			expectedErrorMsg: "invalid structured JSON path structure",
		},
		{
			name:             "empty topic ID",
			filePath:         filepath.Join("my", "data", "structured-json", "subforum-abc", "topic-.json"),
			wantErr:          true,
			expectedErrorMsg: "empty ID component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBasePath, gotSubForumID, gotTopicID, err := ParseStructuredJSONPath(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStructuredJSONPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.expectedErrorMsg != "" && (err == nil || !strings.Contains(err.Error(), tt.expectedErrorMsg)) {
					t.Errorf("ParseStructuredJSONPath() error = %v, expected to contain %s", err, tt.expectedErrorMsg)
				}
				return
			}
			if gotBasePath != tt.wantBasePath {
				t.Errorf("ParseStructuredJSONPath() gotBasePath = %v, want %v", gotBasePath, tt.wantBasePath)
			}
			if gotSubForumID != tt.wantSubForumID {
				t.Errorf("ParseStructuredJSONPath() gotSubForumID = %v, want %v", gotSubForumID, tt.wantSubForumID)
			}
			if gotTopicID != tt.wantTopicID {
				t.Errorf("ParseStructuredJSONPath() gotTopicID = %v, want %v", gotTopicID, tt.wantTopicID)
			}
		})
	}
}

func TestParseSubForumMetadataIndexPath(t *testing.T) {
	tests := []struct {
		name             string
		filePath         string
		wantBasePath     string
		wantSubForumID   string
		wantErr          bool
		expectedErrorMsg string
	}{
		{
			name:           "valid path",
			filePath:       filepath.Join("root", "archive", "metadata", "subforum-xyz", "index.json"),
			wantBasePath:   filepath.Join("root", "archive"),
			wantSubForumID: "xyz",
			wantErr:        false,
		},
		{
			name:             "invalid structure - wrong filename",
			filePath:         filepath.Join("root", "archive", "metadata", "subforum-xyz", "data.json"),
			wantErr:          true,
			expectedErrorMsg: "invalid metadata index path structure",
		},
		{
			name:             "empty subforum ID",
			filePath:         filepath.Join("root", "archive", "metadata", "subforum-", "index.json"),
			wantErr:          true,
			expectedErrorMsg: "empty ID component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBasePath, gotSubForumID, err := ParseSubForumMetadataIndexPath(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSubForumMetadataIndexPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.expectedErrorMsg != "" && (err == nil || !strings.Contains(err.Error(), tt.expectedErrorMsg)) {
					t.Errorf("ParseSubForumMetadataIndexPath() error = %v, expected to contain %s", err, tt.expectedErrorMsg)
				}
				return
			}
			if gotBasePath != tt.wantBasePath {
				t.Errorf("ParseSubForumMetadataIndexPath() gotBasePath = %v, want %v", gotBasePath, tt.wantBasePath)
			}
			if gotSubForumID != tt.wantSubForumID {
				t.Errorf("ParseSubForumMetadataIndexPath() gotSubForumID = %v, want %v", gotSubForumID, tt.wantSubForumID)
			}
		})
	}
}

func TestReadSubForumMetadata(t *testing.T) {
	basePath, err := os.MkdirTemp("", "read_metadata_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath)

	if err := InitializeStorage(basePath); err != nil { // Ensures metadata dir parent exists for WriteSubForumMetadata
		t.Fatalf("InitializeStorage failed: %v", err)
	}

	subForumID := "testSF"

	// Case 1: Valid metadata file
	validMeta := SubForumMetadata{
		TotalTopics:         50,
		PagesPerTopic:       map[string]int{"t1": 1, "t2": 2},
		LastUpdateTimestamp: "2023-10-27T10:00:00Z",
	}
	if err := WriteSubForumMetadata(basePath, subForumID, validMeta); err != nil {
		t.Fatalf("WriteSubForumMetadata for valid case failed: %v", err)
	}
	readMeta, err := ReadSubForumMetadata(basePath, subForumID)
	if err != nil {
		t.Errorf("ReadSubForumMetadata failed for valid case: %v", err)
	}
	if readMeta.TotalTopics != validMeta.TotalTopics { // Basic check, can use reflect.DeepEqual for full struct
		t.Errorf("ReadSubForumMetadata content mismatch. Got %+v, expected %+v", readMeta, validMeta)
	}

	// Case 2: Invalid TotalTopics
	invalidTopicsMeta := SubForumMetadata{TotalTopics: -1}
	if err := WriteSubForumMetadata(basePath, subForumID, invalidTopicsMeta); err != nil {
		t.Fatalf("WriteSubForumMetadata for invalid topics failed: %v", err)
	}
	// Expect ErrStorageInvalidFormat
	_, err = ReadSubForumMetadata(basePath, subForumID)
	if !errors.Is(err, ErrStorageInvalidFormat) {
		t.Errorf("ReadSubForumMetadata wrong error for negative TotalTopics. Got %T, %v, want wrapping %T", err, err, ErrStorageInvalidFormat)
	}

	// Case 3: Non-existent metadata file
	metadataFilePath := GetSubForumMetadataIndexPath(basePath, subForumID)
	if err := os.Remove(metadataFilePath); err != nil {
		t.Fatalf("Failed to remove metadata file for non-existent test: %v", err)
	}
	// Expect ErrStorageNotFound
	_, err = ReadSubForumMetadata(basePath, subForumID)
	if !errors.Is(err, ErrStorageNotFound) {
		t.Errorf("ReadSubForumMetadata wrong error for non-existent file. Got %T, %v, want wrapping %T", err, err, ErrStorageNotFound)
	}

	// Case 4: Malformed JSON
	// Create the directory for the metadata file first
	metaDir := filepath.Dir(metadataFilePath)
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		t.Fatalf("Failed to create directory for malformed JSON test: %v", err)
	}
	malformedJSON := []byte("{\"TotalTopics\": 10, \"LastUpdateTimestamp\": \"bad-time\"THIS_IS_BAD}")
	if err := os.WriteFile(metadataFilePath, malformedJSON, 0644); err != nil {
		t.Fatalf("Failed to write malformed JSON metadata: %v", err)
	}
	// Expect ErrStorageInvalidFormat for unmarshal error
	_, err = ReadSubForumMetadata(basePath, subForumID)
	if !errors.Is(err, ErrStorageInvalidFormat) {
		t.Errorf("ReadSubForumMetadata wrong error for malformed JSON. Got %T, %v, want wrapping %T", err, err, ErrStorageInvalidFormat)
	}
}

func TestGetDirectorySize(t *testing.T) {
	basePath, err := os.MkdirTemp("", "dirsize_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath)

	// Create some structure and files
	subDir1 := filepath.Join(basePath, "sub1")
	if err := os.Mkdir(subDir1, 0755); err != nil {
		t.Fatalf("Failed to create subDir1: %v", err)
	}
	subDir2 := filepath.Join(basePath, "sub2")
	if err := os.Mkdir(subDir2, 0755); err != nil {
		t.Fatalf("Failed to create subDir2: %v", err)
	}

	file1Size := int64(100)
	file2Size := int64(250)
	file3Size := int64(50)

	if err := os.WriteFile(filepath.Join(basePath, "file1.txt"), make([]byte, file1Size), 0644); err != nil {
		t.Fatalf("Failed to write file1.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir1, "file2.txt"), make([]byte, file2Size), 0644); err != nil {
		t.Fatalf("Failed to write file2.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir2, "file3.txt"), make([]byte, file3Size), 0644); err != nil {
		t.Fatalf("Failed to write file3.txt: %v", err)
	}

	// Case 1: Get size of the basePath
	expectedTotalSize := file1Size + file2Size + file3Size
	actualTotalSize, err := GetDirectorySize(basePath)
	if err != nil {
		t.Errorf("GetDirectorySize for basePath failed: %v", err)
	}
	if actualTotalSize != expectedTotalSize {
		t.Errorf("GetDirectorySize for basePath: got %d, want %d", actualTotalSize, expectedTotalSize)
	}

	// Case 2: Get size of a sub-directory
	expectedSubDir1Size := file2Size
	actualSubDir1Size, err := GetDirectorySize(subDir1)
	if err != nil {
		t.Errorf("GetDirectorySize for subDir1 failed: %v", err)
	}
	if actualSubDir1Size != expectedSubDir1Size {
		t.Errorf("GetDirectorySize for subDir1: got %d, want %d", actualSubDir1Size, expectedSubDir1Size)
	}

	// Case 3: Get size of an empty directory
	emptyDir := filepath.Join(basePath, "emptySub")
	if err := os.Mkdir(emptyDir, 0755); err != nil {
		t.Fatalf("Failed to create emptyDir: %v", err)
	}
	actualEmptyDirSize, err := GetDirectorySize(emptyDir)
	if err != nil {
		t.Errorf("GetDirectorySize for emptyDir failed: %v", err)
	}
	if actualEmptyDirSize != 0 {
		t.Errorf("GetDirectorySize for emptyDir: got %d, want 0", actualEmptyDirSize)
	}

	// Case 4: Non-existent directory
	_, err = GetDirectorySize(filepath.Join(basePath, "nonexistent"))
	if err == nil {
		t.Errorf("GetDirectorySize should have failed for a non-existent directory")
	} else {
		t.Logf("GetDirectorySize failed as expected for non-existent path: %v", err) // Log for info
	}
}

func TestCheckStorageQuota(t *testing.T) {
	basePath, err := os.MkdirTemp("", "quota_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(basePath)

	// Helper to capture log output
	var logBuf bytes.Buffer
	originalInfoLog, originalWarnLog, originalErrorLog := infoLog, warnLog, errorLog // Save original loggers
	SetLoggerOutput(&logBuf, &logBuf, &logBuf)                                       // Redirect all to logBuf
	defer func() {
		infoLog, warnLog, errorLog = originalInfoLog, originalWarnLog, originalErrorLog // Restore original loggers
	}()

	// Create a file to have some size
	fileSize := int64(1000)
	if err := os.WriteFile(filepath.Join(basePath, "testfile.dat"), make([]byte, fileSize), 0644); err != nil {
		t.Fatalf("Failed to write testfile.dat: %v", err)
	}

	tests := []struct {
		name                     string
		basePath                 string
		warningThreshold         float64
		quotaBytes               int64
		wantCurrentUsageBytes    int64   // Can be inferred if basePath is always the same, but good for clarity
		wantUsagePercentageRough float64 // Approximate due to float precision
		wantIsWarning            bool
		wantError                bool
		logShouldContain         string // This will be less directly used, logic will be more specific
	}{
		{
			name:                  "no quota",
			basePath:              basePath,
			warningThreshold:      80.0,
			quotaBytes:            0,
			wantCurrentUsageBytes: fileSize,
			wantIsWarning:         false,
			wantError:             false,
			logShouldContain:      "INFO: storage: No quota set", // Adjusted for new check logic
		},
		{
			name:                     "usage below threshold",
			basePath:                 basePath,
			warningThreshold:         80.0,
			quotaBytes:               2000,
			wantCurrentUsageBytes:    fileSize,
			wantUsagePercentageRough: 50.0,
			wantIsWarning:            false,
			wantError:                false,
			logShouldContain:         "INFO: storage: Storage usage for", // Adjusted
		},
		{
			name:                     "usage at threshold",
			basePath:                 basePath,
			warningThreshold:         50.0,
			quotaBytes:               2000,
			wantCurrentUsageBytes:    fileSize,
			wantUsagePercentageRough: 50.0,
			wantIsWarning:            true,
			wantError:                false,
			logShouldContain:         "WARN: storage: Storage usage for", // Adjusted
		},
		{
			name:                     "usage above threshold",
			basePath:                 basePath,
			warningThreshold:         40.0,
			quotaBytes:               2000,
			wantCurrentUsageBytes:    fileSize,
			wantUsagePercentageRough: 50.0,
			wantIsWarning:            true,
			wantError:                false,
			logShouldContain:         "WARN: storage: Storage usage for", // Adjusted
		},
		{
			name:                  "error getting size",
			basePath:              filepath.Join(basePath, "nonexistent"),
			warningThreshold:      80.0,
			quotaBytes:            1000,
			wantCurrentUsageBytes: 0,
			wantIsWarning:         false,
			wantError:             true,
			logShouldContain:      "ERROR: storage: Error checking storage quota", // Adjusted
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logBuf.Reset()
			status := CheckStorageQuota(tt.basePath, tt.warningThreshold, tt.quotaBytes)

			if (status.Error != nil) != tt.wantError {
				t.Errorf("CheckStorageQuota() error = %v, wantError %v", status.Error, tt.wantError)
				return
			}

			logStr := logBuf.String()
			if tt.wantError {
				if !strings.HasPrefix(logStr, "ERROR: storage:") || !strings.Contains(logStr, "Error checking storage quota") {
					t.Errorf("CheckStorageQuota() error log = %s, want prefix ERROR: storage: and content 'Error checking storage quota'", logStr)
				}
				return
			}

			if tt.name == "no quota" {
				if !strings.HasPrefix(logStr, "INFO: storage:") || !strings.Contains(logStr, "No quota set") {
					t.Errorf("CheckStorageQuota() 'no quota' log = %s, want prefix INFO: storage: and content 'No quota set'", logStr)
				}
			} else if tt.wantIsWarning {
				if !strings.HasPrefix(logStr, "WARN: storage:") || !strings.Contains(logStr, "Storage usage for") {
					t.Errorf("CheckStorageQuota() warning log = %s, want prefix WARN: storage: and content 'Storage usage for'", logStr)
				}
			} else {
				if !strings.HasPrefix(logStr, "INFO: storage:") || !strings.Contains(logStr, "Storage usage for") {
					t.Errorf("CheckStorageQuota() info log = %s, want prefix INFO: storage: and content 'Storage usage for'", logStr)
				}
			}

			if status.CurrentUsageBytes != tt.wantCurrentUsageBytes {
				t.Errorf("CheckStorageQuota() CurrentUsageBytes = %d, want %d", status.CurrentUsageBytes, tt.wantCurrentUsageBytes)
			}
			if tt.quotaBytes > 0 {
				diff := status.UsagePercentage - tt.wantUsagePercentageRough
				if diff < -0.01 || diff > 0.01 {
					t.Errorf("CheckStorageQuota() UsagePercentage = %.2f, want roughly %.2f", status.UsagePercentage, tt.wantUsagePercentageRough)
				}
			}
			if status.IsWarning != tt.wantIsWarning {
				t.Errorf("CheckStorageQuota() IsWarning = %v, want %v", status.IsWarning, tt.wantIsWarning)
			}
			// logShouldContain field is now handled by the more specific checks above
		})
	}
}

func TestBackupMetadataAndProgress(t *testing.T) {
	archiveBasePathMaster, err := os.MkdirTemp("", "archive_base_master_")
	if err != nil {
		t.Fatalf("Failed to create archive_base_master_: %v", err)
	}
	defer os.RemoveAll(archiveBasePathMaster)

	// Helper to setup initial archive state in a *copy* of the master path for true isolation per sub-test
	setupInitialArchive := func(t *testing.T, createProgress, createMetadata bool) string {
		tempArchiveBasePath, err := os.MkdirTemp("", "archive_test_instance_")
		if err != nil {
			t.Fatalf("Failed to create tempArchiveBasePath: %v", err)
		}

		if err := InitializeStorage(tempArchiveBasePath); err != nil {
			_ = os.RemoveAll(tempArchiveBasePath) // Attempt to clean up on error
			t.Fatalf("InitializeStorage failed during test setup on %s: %v", tempArchiveBasePath, err)
		}

		if createProgress {
			// progress.json is already created by InitializeStorage, potentially update it if needed
			// For this test, default initial progress is fine.
		} else {
			if err := os.Remove(filepath.Join(tempArchiveBasePath, "progress.json")); err != nil && !os.IsNotExist(err) {
				_ = os.RemoveAll(tempArchiveBasePath)
				t.Fatalf("Failed to remove progress.json from %s: %v", tempArchiveBasePath, err)
			}
		}

		if createMetadata {
			metaSF1 := SubForumMetadata{TotalTopics: 1, PagesPerTopic: map[string]int{"t1": 1}, LastUpdateTimestamp: "ts1"}
			if err := WriteSubForumMetadata(tempArchiveBasePath, "sf1", metaSF1); err != nil {
				_ = os.RemoveAll(tempArchiveBasePath)
				t.Fatalf("Failed to write sf1 metadata to %s: %v", tempArchiveBasePath, err)
			}
			nestedMetaDir := filepath.Join(tempArchiveBasePath, "metadata", "subforum-sf1", "nested")
			if err := os.MkdirAll(nestedMetaDir, 0755); err != nil {
				_ = os.RemoveAll(tempArchiveBasePath)
				t.Fatalf("Failed to create nested metadata dir in %s: %v", tempArchiveBasePath, err)
			}
			if err := os.WriteFile(filepath.Join(nestedMetaDir, "test.txt"), []byte("hello"), 0644); err != nil {
				_ = os.RemoveAll(tempArchiveBasePath)
				t.Fatalf("Failed to write nested metadata file in %s: %v", tempArchiveBasePath, err)
			}
		} else {
			if err := os.RemoveAll(filepath.Join(tempArchiveBasePath, "metadata")); err != nil && !os.IsNotExist(err) {
				_ = os.RemoveAll(tempArchiveBasePath)
				t.Fatalf("Failed to remove metadata dir from %s: %v", tempArchiveBasePath, err)
			}
		}
		return tempArchiveBasePath // Return the path to be cleaned up by the sub-test
	}

	var logBuf bytes.Buffer
	originalInfoLog, originalWarnLog, originalErrorLog := infoLog, warnLog, errorLog
	SetLoggerOutput(&logBuf, &logBuf, &logBuf)
	defer func() {
		infoLog, warnLog, errorLog = originalInfoLog, originalWarnLog, originalErrorLog
	}()

	t.Run("backup with all data", func(t *testing.T) {
		currentTestArchiveBasePath := setupInitialArchive(t, true, true)
		defer os.RemoveAll(currentTestArchiveBasePath)
		currentTestBackupParentDir, _ := os.MkdirTemp("", "bp_alldata_")
		defer os.RemoveAll(currentTestBackupParentDir)
		logBuf.Reset()

		backupPath, err := BackupMetadataAndProgress(currentTestArchiveBasePath, currentTestBackupParentDir)
		if err != nil {
			t.Fatalf("BackupMetadataAndProgress failed: %v", err)
		}
		if !strings.Contains(logBuf.String(), "BackupMetadataAndProgress finished successfully") {
			t.Errorf("Expected success log, got: %s", logBuf.String())
		}

		t.Logf("[AFTER BACKUP - all data case] Backup created at: %s", backupPath)
		t.Logf("[AFTER BACKUP - all data case] Listing backupPath (%s):", backupPath)
		_ = filepath.WalkDir(backupPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Backup Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		originalProgress, _ := ReadProgressFile(currentTestArchiveBasePath)
		backedUpProgress, err := ReadProgressFile(backupPath)
		if err != nil {
			t.Errorf("Failed to read backed up progress.json from %s: %v", backupPath, err)
		}
		if backedUpProgress != originalProgress {
			t.Errorf("Backed up progress.json content mismatch. Got %+v, expected %+v", backedUpProgress, originalProgress)
		}
		backedUpMetaSF1Path := GetSubForumMetadataIndexPath(backupPath, "sf1")
		if _, err := os.Stat(backedUpMetaSF1Path); os.IsNotExist(err) {
			t.Errorf("Backed up metadata for sf1 not found at %s", backedUpMetaSF1Path)
		}
		backedUpNestedFile := filepath.Join(backupPath, "metadata", "subforum-sf1", "nested", "test.txt")
		if _, err := os.Stat(backedUpNestedFile); os.IsNotExist(err) {
			t.Errorf("Backed up nested metadata file not found at %s", backedUpNestedFile)
		}
	})

	t.Run("backup with missing progress.json", func(t *testing.T) {
		currentTestArchiveBasePath := setupInitialArchive(t, false, true)
		defer os.RemoveAll(currentTestArchiveBasePath)
		currentTestBackupParentDir, _ := os.MkdirTemp("", "bp_missprog_")
		defer os.RemoveAll(currentTestBackupParentDir)
		logBuf.Reset()

		t.Logf("[BEFORE BACKUP - missing progress.json case] Listing archiveBasePath (%s):", currentTestArchiveBasePath)
		_ = filepath.WalkDir(currentTestArchiveBasePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Source Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		backupPath, err := BackupMetadataAndProgress(currentTestArchiveBasePath, currentTestBackupParentDir)
		if err != nil {
			t.Fatalf("BackupMetadataAndProgress failed: %v", err)
		}
		t.Logf("[AFTER BACKUP - missing progress.json case] Backup created at: %s", backupPath)
		t.Logf("[AFTER BACKUP - missing progress.json case] Listing backupPath (%s):", backupPath)
		_ = filepath.WalkDir(backupPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Backup Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		progressInBackupPath := filepath.Join(backupPath, "progress.json")
		if _, errStat := os.Stat(progressInBackupPath); !os.IsNotExist(errStat) {
			t.Errorf("progress.json should not exist in backup at %s when source is missing, but it does (stat error: %v).", progressInBackupPath, errStat)
		}
		backedUpMetaSF1Path := GetSubForumMetadataIndexPath(backupPath, "sf1")
		if _, err := os.Stat(backedUpMetaSF1Path); os.IsNotExist(err) {
			t.Errorf("Backed up metadata for sf1 not found at %s when progress was missing", backedUpMetaSF1Path)
		}
	})

	t.Run("backup with missing metadata dir", func(t *testing.T) {
		currentTestArchiveBasePath := setupInitialArchive(t, true, false)
		defer os.RemoveAll(currentTestArchiveBasePath)
		currentTestBackupParentDir, _ := os.MkdirTemp("", "bp_missmeta_")
		defer os.RemoveAll(currentTestBackupParentDir)
		logBuf.Reset()

		t.Logf("[BEFORE BACKUP - missing metadata dir case] Listing archiveBasePath (%s):", currentTestArchiveBasePath)
		_ = filepath.WalkDir(currentTestArchiveBasePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Source Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		backupPath, err := BackupMetadataAndProgress(currentTestArchiveBasePath, currentTestBackupParentDir)
		if err != nil {
			t.Fatalf("BackupMetadataAndProgress failed: %v", err)
		}
		t.Logf("[AFTER BACKUP - missing metadata dir case] Backup created at: %s", backupPath)
		t.Logf("[AFTER BACKUP - missing metadata dir case] Listing backupPath (%s):", backupPath)
		_ = filepath.WalkDir(backupPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Backup Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		metadataInBackupPath := filepath.Join(backupPath, "metadata")
		if _, errStat := os.Stat(metadataInBackupPath); !os.IsNotExist(errStat) {
			t.Errorf("metadata directory should not exist in backup at %s when source is missing, but it does (stat error: %v).", metadataInBackupPath, errStat)
		}
		if _, err := ReadProgressFile(backupPath); err != nil {
			t.Errorf("Failed to read backed up progress.json from %s when metadata was missing: %v", backupPath, err)
		}
	})

	t.Run("backup with both missing", func(t *testing.T) {
		currentTestArchiveBasePath := setupInitialArchive(t, false, false)
		defer os.RemoveAll(currentTestArchiveBasePath)
		currentTestBackupParentDir, _ := os.MkdirTemp("", "bp_bothmiss_")
		defer os.RemoveAll(currentTestBackupParentDir)
		logBuf.Reset()

		t.Logf("[BEFORE BACKUP - both missing case] Listing archiveBasePath (%s):", currentTestArchiveBasePath)
		_ = filepath.WalkDir(currentTestArchiveBasePath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Source Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		backupPath, err := BackupMetadataAndProgress(currentTestArchiveBasePath, currentTestBackupParentDir)
		if err != nil {
			t.Fatalf("BackupMetadataAndProgress failed: %v", err)
		}
		t.Logf("[AFTER BACKUP - both missing case] Backup created at: %s", backupPath)
		t.Logf("[AFTER BACKUP - both missing case] Listing backupPath (%s):", backupPath)
		_ = filepath.WalkDir(backupPath, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			t.Logf("  Backup Path: %s (IsDir: %v)", path, d.IsDir())
			return nil
		})

		progressInBackupPath := filepath.Join(backupPath, "progress.json")
		if _, errStat := os.Stat(progressInBackupPath); !os.IsNotExist(errStat) {
			t.Errorf("progress.json should not exist in backup at %s when source is missing (both case, stat err: %v).", progressInBackupPath, errStat)
		}
		metadataInBackupPath := filepath.Join(backupPath, "metadata")
		if _, errStat := os.Stat(metadataInBackupPath); !os.IsNotExist(errStat) {
			t.Errorf("metadata directory should not exist in backup at %s when source is missing (both case, stat err: %v).", metadataInBackupPath, errStat)
		}
		if _, errStat := os.Stat(backupPath); os.IsNotExist(errStat) {
			t.Errorf("Backup directory %s was not created when both sources missing.", backupPath)
		}
	})
}

func TestListBackups(t *testing.T) {
	backupBaseDir, err := os.MkdirTemp("", "list_backups_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir for ListBackups: %v", err)
	}
	defer os.RemoveAll(backupBaseDir)

	var logBuf bytes.Buffer
	originalInfoLog, originalWarnLog, originalErrorLog := infoLog, warnLog, errorLog
	SetLoggerOutput(&logBuf, &logBuf, &logBuf)
	defer func() {
		infoLog, warnLog, errorLog = originalInfoLog, originalWarnLog, originalErrorLog
	}()

	// Case 1: Backup directory doesn't exist (or is empty)
	// For non-existent, create then remove to test that path
	nonExistentBackupDir := filepath.Join(backupBaseDir, "nonexistent")
	backups, err := ListBackups(nonExistentBackupDir)
	if err != nil {
		t.Errorf("ListBackups failed for non-existent dir: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("Expected 0 backups for non-existent dir, got %d", len(backups))
	}

	// For empty, just call on the created backupBaseDir
	backups, err = ListBackups(backupBaseDir)
	if err != nil {
		t.Errorf("ListBackups failed for empty dir: %v", err)
	}
	if len(backups) != 0 {
		t.Errorf("Expected 0 backups for empty dir, got %d", len(backups))
	}

	// Case 2: Create some mock backup directories
	// Timestamps for consistent sorting and checking
	ts1Str := "20230101100000"
	ts2Str := "20230102120000"
	ts3StrInvalid := "project-waypoint-backup-badtimestamp"
	ts4StrNotDir := "project-waypoint-backup-20230103140000.txt" // A file, not a dir

	backup1Dir := filepath.Join(backupBaseDir, "project-waypoint-backup-"+ts1Str)
	if err := os.Mkdir(backup1Dir, 0755); err != nil {
		t.Fatalf("Failed to create backup1Dir: %v", err)
	}

	backup2Dir := filepath.Join(backupBaseDir, "project-waypoint-backup-"+ts2Str)
	if err := os.Mkdir(backup2Dir, 0755); err != nil {
		t.Fatalf("Failed to create backup2Dir: %v", err)
	}

	invalidBackupDir := filepath.Join(backupBaseDir, ts3StrInvalid) // Name doesn't have full prefix or bad ts
	if err := os.Mkdir(invalidBackupDir, 0755); err != nil {
		t.Fatalf("Failed to create invalidBackupDir: %v", err)
	}

	notADirPath := filepath.Join(backupBaseDir, ts4StrNotDir)
	if err := os.WriteFile(notADirPath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create notADirPath file: %v", err)
	}

	// Directory with correct prefix but unparseable timestamp
	backupBadTsDir := filepath.Join(backupBaseDir, "project-waypoint-backup-20231301100000") // Invalid month
	if err := os.Mkdir(backupBadTsDir, 0755); err != nil {
		t.Fatalf("Failed to create backupBadTsDir: %v", err)
	}

	logBuf.Reset()
	backups, err = ListBackups(backupBaseDir)
	if err != nil {
		t.Fatalf("ListBackups failed with actual backups: %v", err)
	}

	if len(backups) != 2 {
		t.Errorf("Expected 2 valid backups, got %d", len(backups))
	}

	if !strings.Contains(logBuf.String(), "invalid timestamp format") {
		t.Errorf("Expected log warning for bad timestamp format, got: %s", logBuf.String())
	}

	// Check sorting (newest first) and content
	ts1Time, _ := time.Parse("20060102150405", ts1Str)
	ts2Time, _ := time.Parse("20060102150405", ts2Str)

	if len(backups) == 2 { // Proceed only if we have 2, to prevent panic
		if backups[0].Timestamp != ts2Time || !strings.HasSuffix(backups[0].Path, ts2Str) {
			t.Errorf("First backup (newest) mismatch. Got TS %v, Path %s. Expected TS %v, Path suffix %s",
				backups[0].Timestamp, backups[0].Path, ts2Time, ts2Str)
		}
		if backups[1].Timestamp != ts1Time || !strings.HasSuffix(backups[1].Path, ts1Str) {
			t.Errorf("Second backup (older) mismatch. Got TS %v, Path %s. Expected TS %v, Path suffix %s",
				backups[1].Timestamp, backups[1].Path, ts1Time, ts1Str)
		}
	}
}
