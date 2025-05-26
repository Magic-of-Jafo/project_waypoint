package indexerlogic

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"waypoint_archive_scripts/pkg/data"
)

func TestReadTopicIndexCSV(t *testing.T) {
	// Create a temporary test CSV file
	tempDir := t.TempDir()
	validCSVPath := filepath.Join(tempDir, "valid_topics.csv")
	emptyCSVPath := filepath.Join(tempDir, "empty_topics.csv")
	headerOnlyCSVPath := filepath.Join(tempDir, "header_only_topics.csv")
	malformedCSVPath := filepath.Join(tempDir, "malformed_topics.csv")
	nonExistentPath := filepath.Join(tempDir, "nonexistent.csv")

	// Valid CSV content
	validContent := `TopicID,Title,URL
topic1,Title One,http://example.com/topic1
topic2,Title Two,http://example.com/topic2
`
	if err := os.WriteFile(validCSVPath, []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to write valid test CSV: %v", err)
	}

	// Empty CSV content (just header)
	headerOnlyContent := `TopicID,Title,URL
`
	if err := os.WriteFile(headerOnlyCSVPath, []byte(headerOnlyContent), 0644); err != nil {
		t.Fatalf("Failed to write header-only test CSV: %v", err)
	}
	// Empty file (0 bytes)
	f, err := os.Create(emptyCSVPath)
	if err != nil {
		t.Fatalf("Failed to create empty test CSV: %v", err)
	}
	f.Close() // Ensure file is closed

	// Malformed CSV (not enough columns)
	malformedContent := `TopicID,Title
topic_bad,Bad Title
`
	if err := os.WriteFile(malformedCSVPath, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("Failed to write malformed test CSV: %v", err)
	}

	subForumID := "sf1"

	tests := []struct {
		name       string
		filePath   string
		subForumID string
		want       []data.Topic
		wantErr    bool
	}{
		{
			name:       "Valid CSV",
			filePath:   validCSVPath,
			subForumID: subForumID,
			want: []data.Topic{
				{ID: "topic1", SubForumID: subForumID, Title: "Title One", URL: "http://example.com/topic1"},
				{ID: "topic2", SubForumID: subForumID, Title: "Title Two", URL: "http://example.com/topic2"},
			},
			wantErr: false,
		},
		{
			name:       "Header only CSV",
			filePath:   headerOnlyCSVPath,
			subForumID: subForumID,
			want:       []data.Topic{},
			wantErr:    false,
		},
		{
			name:       "Empty file CSV", // Test case for completely empty file
			filePath:   emptyCSVPath,
			subForumID: subForumID,
			want:       []data.Topic{}, // Corrected: Expect empty slice, no error for this case now
			wantErr:    false,          // Corrected: Expect no error
		},
		{
			name:       "Malformed CSV",
			filePath:   malformedCSVPath,
			subForumID: subForumID,
			want:       nil,
			wantErr:    true,
		},
		{
			name:       "File not found",
			filePath:   nonExistentPath,
			subForumID: subForumID,
			want:       nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadTopicIndexCSV(tt.filePath, tt.subForumID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadTopicIndexCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				// For empty file, expected 'got' is []data.Topic{} if no error, but we expect error.
				// So this check is fine as is.
				t.Errorf("ReadTopicIndexCSV() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadSubForumListCSV(t *testing.T) {
	tempDir := t.TempDir()
	validCSVPath := filepath.Join(tempDir, "valid_subforums.csv")
	emptyCSVPath := filepath.Join(tempDir, "empty_subforums.csv")
	headerOnlyCSVPath := filepath.Join(tempDir, "header_only_subforums.csv")
	malformedCSVPath := filepath.Join(tempDir, "malformed_subforums.csv")
	nonExistentPath := filepath.Join(tempDir, "nonexistent_sf.csv")

	validContent := `SubForumID,SubForumName
sf1,SubForum One
sf2,SubForum Two
`
	if err := os.WriteFile(validCSVPath, []byte(validContent), 0644); err != nil {
		t.Fatalf("Failed to write valid subforum test CSV: %v", err)
	}

	headerOnlyContent := `SubForumID,SubForumName
`
	if err := os.WriteFile(headerOnlyCSVPath, []byte(headerOnlyContent), 0644); err != nil {
		t.Fatalf("Failed to write header-only subforum test CSV: %v", err)
	}

	f, err := os.Create(emptyCSVPath)
	if err != nil {
		t.Fatalf("Failed to create empty subforum test CSV: %v", err)
	}
	f.Close() // Ensure file is closed

	malformedContent := `SubForumID
sf_bad
`
	if err := os.WriteFile(malformedCSVPath, []byte(malformedContent), 0644); err != nil {
		t.Fatalf("Failed to write malformed subforum test CSV: %v", err)
	}

	tests := []struct {
		name     string
		filePath string
		want     map[string]string
		wantErr  bool
	}{
		{
			name:     "Valid CSV",
			filePath: validCSVPath,
			want: map[string]string{
				"sf1": "SubForum One",
				"sf2": "SubForum Two",
			},
			wantErr: false,
		},
		{
			name:     "Header only CSV",
			filePath: headerOnlyCSVPath,
			want:     map[string]string{},
			wantErr:  false,
		},
		{
			name:     "Empty file CSV",
			filePath: emptyCSVPath,
			want:     map[string]string{}, // Corrected: Expect empty map, no error
			wantErr:  false,               // Corrected: Expect no error
		},
		{
			name:     "Malformed CSV",
			filePath: malformedCSVPath,
			want:     nil,
			wantErr:  true,
		},
		{
			name:     "File not found",
			filePath: nonExistentPath,
			want:     nil,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadSubForumListCSV(tt.filePath)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadSubForumListCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadSubForumListCSV() got = %v, want %v", got, tt.want)
			}
		})
	}
}
