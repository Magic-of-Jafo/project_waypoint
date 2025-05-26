package indexerlogic

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"waypoint_archive_scripts/pkg/data"
)

func TestReadTopicIndexCSV_Success(t *testing.T) {
	content := `TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked,SubForumID

t1,Test Topic 1,http://example.com/t1,user1,10,100,user2,2023-01-01T12:00:00Z,false,false,sf1

t2,Test Topic 2,http://example.com/t2,user3,5,50,user4,2023-01-02T12:00:00Z,true,true,sf1`
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "topic_index_sf1.csv")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	want := []data.Topic{
		{ID: "t1", Title: "Test Topic 1", URL: "http://example.com/t1", SubForumID: "sf1", AuthorUsername: "user1", Replies: 10, Views: 100, LastPostUsername: "user2", LastPostTimestampRaw: "2023-01-01T12:00:00Z", IsSticky: false, IsLocked: false},
		{ID: "t2", Title: "Test Topic 2", URL: "http://example.com/t2", SubForumID: "sf1", AuthorUsername: "user3", Replies: 5, Views: 50, LastPostUsername: "user4", LastPostTimestampRaw: "2023-01-02T12:00:00Z", IsSticky: true, IsLocked: true},
	}

	got, err := ReadTopicIndexCSV(filePath, "sf1")
	if err != nil {
		t.Errorf("ReadTopicIndexCSV() error = %v, wantErr nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadTopicIndexCSV() got = %v, want %v", got, want)
	}
}

func TestReadTopicIndexCSV_FileNotExist(t *testing.T) {
	_, err := ReadTopicIndexCSV("nonexistent.csv", "sf1")
	if err == nil {
		t.Errorf("ReadTopicIndexCSV() error = nil, wantErr true for non-existent file")
	}
}

func TestReadTopicIndexCSV_MalformedCSV(t *testing.T) {
	content := `TopicID,Title\nMalformedRow` // Missing URL and other fields
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "malformed.csv")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// We expect an error because the row doesn't have enough fields.
	// The function tries to parse specific indices.
	// It should log an error and skip the row, but the test expects an empty slice and NO error if all rows are skipped.
	// Let's refine this: if there's a header but no valid data rows, it should return an empty slice and no error.
	// If the CSV itself is so malformed it cannot be read (e.g. unclosed quotes), csv.ReadAll returns an error.

	// Current ReadTopicIndexCSV skips malformed rows and logs. So, no error returned, empty slice.
	got, err := ReadTopicIndexCSV(filePath, "sf1")
	if err != nil {
		t.Errorf("ReadTopicIndexCSV() with malformed data error = %v, want nil (skips row)", err)
	}
	if len(got) != 0 {
		t.Errorf("ReadTopicIndexCSV() with malformed data got %d topics, want 0", len(got))
	}
}

func TestReadSubForumListCSV_Success(t *testing.T) {
	content := `SubForumID,SubForumName,SubForumURL
sf1,SubForum One,http://example.com/sf1
sf2,SubForum Two,http://example.com/sf2
` // Added SubForumURL column
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "subforums.csv")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	want := map[string]SubForumNameAndURL{
		"sf1": {Name: "SubForum One", URL: "http://example.com/sf1"},
		"sf2": {Name: "SubForum Two", URL: "http://example.com/sf2"},
	}
	got, err := ReadSubForumListCSV(filePath)
	if err != nil {
		t.Errorf("ReadSubForumListCSV() error = %v, wantErr nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ReadSubForumListCSV() got = %v, want %v", got, want)
	}
}

func TestReadTopicIndexCSV_VariedColumns(t *testing.T) {
	// This test covers cases where optional columns might be missing or empty
	// and ensures parsing is still robust.
	// `AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked,SubForumID` are required for parsing in ReadTopicIndexCSV
	// `TopicID,Title,URL` are essential
	// SubForumID is passed as an argument, not read from CSV in ReadTopicIndexCSV

	tests := []struct {
		name       string
		content    string
		subForumID string
		want       []data.Topic
		wantErr    bool
	}{
		{
			name: "missing optional fields, present required",
			content: `TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked
` + // Removed SubForumID from header
				`t1,Topic,http://url.com,author,10,100,lastposter,2024-01-01T00:00:00Z,false,false`, // SubForumID is not in CSV for this function, it's an arg
			subForumID: "sf_test",
			want: []data.Topic{
				{ID: "t1", Title: "Topic", URL: "http://url.com", AuthorUsername: "author", Replies: 10, Views: 100, LastPostUsername: "lastposter", LastPostTimestampRaw: "2024-01-01T00:00:00Z", IsSticky: false, IsLocked: false, SubForumID: "sf_test"},
			},
			wantErr: false,
		},
		{
			name:       "empty file",
			content:    "",
			subForumID: "sf_empty",
			want:       []data.Topic{}, // Expect empty slice for empty file
			wantErr:    false,          // Changed from true: ReadTopicIndexCSV returns nil error for empty/header-only file
		},
		{
			name:       "header only",
			content:    "TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked,SubForumID",
			subForumID: "sf_header",
			want:       []data.Topic{}, // Expect empty slice for header-only file
			wantErr:    false,
		},
		{
			name: "malformed replies (non-integer)",
			content: `TopicID,Title,URL,AuthorUsername,Replies,Views,LastPostUsername,LastPostTimestamp,IsSticky,IsLocked
` +
				`t1,T,http://u,a,BAD,100,lp,2024-01-01T00:00:00Z,f,f`,
			subForumID: "sf_bad_replies",
			want:       []data.Topic{ // Should skip the row with bad Replies
				// Expect empty because the single row is skipped
			},
			wantErr: false, // The function logs and skips, no error returned to caller for bad row data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			filePath := filepath.Join(tempDir, "topics.csv")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}
			got, err := ReadTopicIndexCSV(filePath, tt.subForumID)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadTopicIndexCSV() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadTopicIndexCSV() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadTopicIndexCSV_HandlesUnclosedQuotes(t *testing.T) {
	content := `TopicID,Title,URL
"t1","Unclosed quote test,http://example.com/t1` // Note the missing closing quote for Title
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "unclosed_quotes.csv")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// csv.Reader.ReadAll() should return an error for unclosed quotes.
	_, err := ReadTopicIndexCSV(filePath, "sf_unclosed")
	if err == nil {
		t.Errorf("ReadTopicIndexCSV() with unclosed quotes expected an error, got nil")
	} else {
		// Check for a generic CSV parsing error, as specific messages can vary.
		if !strings.Contains(strings.ToLower(err.Error()), "parse error") && !strings.Contains(strings.ToLower(err.Error()), "unclosed quoted field") && !strings.Contains(strings.ToLower(err.Error()), "extraneous or missing") {
			t.Errorf("ReadTopicIndexCSV() expected a CSV parse error (e.g., unclosed quote, extraneous quote), got: %v", err)
		}
	}
}

func TestReadSubForumListCSV_FileNotExist(t *testing.T) {
	_, err := ReadSubForumListCSV("nonexistent.csv")
	if err == nil {
		t.Errorf("ReadSubForumListCSV() error = nil, wantErr true for non-existent file")
	}
}

func TestReadSubForumListCSV_MalformedCSV(t *testing.T) {
	// Test with insufficient columns (e.g., missing URL)
	contentMissingColumn := "SubForumID,SubForumName\nsf1,Only Name"
	tempDir := t.TempDir()
	filePathMissingCol := filepath.Join(tempDir, "malformed_subforums_missing_col.csv")
	if err := os.WriteFile(filePathMissingCol, []byte(contentMissingColumn), 0644); err != nil {
		t.Fatalf("Failed to write test file (missing col): %v", err)
	}

	// ReadSubForumListCSV now logs a warning and skips the row, it does not return an error for this case.
	// It would return an empty map if all rows are skipped.
	got, err := ReadSubForumListCSV(filePathMissingCol)
	if err != nil {
		t.Errorf("ReadSubForumListCSV() with missing column data error = %v, want nil (should skip row)", err)
	}
	if len(got) != 0 { // Expect that the malformed row was skipped
		t.Errorf("ReadSubForumListCSV() with missing column data got %d entries, want 0", len(got))
	}

	// Test with a more fundamentally malformed CSV (e.g., unclosed quote) that causes a CSV parse error
	contentUnclosedQuote := `SubForumID,SubForumName,SubForumURL
sf1,"Unclosed Name,http://example.com/sf1`
	filePathUnclosedQuote := filepath.Join(tempDir, "malformed_subforums_unclosed.csv")
	if err := os.WriteFile(filePathUnclosedQuote, []byte(contentUnclosedQuote), 0644); err != nil {
		t.Fatalf("Failed to write test file (unclosed quote): %v", err)
	}

	_, errUnclosed := ReadSubForumListCSV(filePathUnclosedQuote)
	if errUnclosed == nil {
		t.Errorf("ReadSubForumListCSV() with unclosed quote data, wantErr true, got nil")
	} else if !strings.Contains(strings.ToLower(errUnclosed.Error()), "parse error") {
		t.Errorf("ReadSubForumListCSV() with unclosed quote data expected a parse error, got: %v", errUnclosed)
	}
}
