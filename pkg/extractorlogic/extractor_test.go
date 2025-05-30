package extractorlogic

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"pkg/data" // Corrected to fully qualified path

	"github.com/PuerkitoBio/goquery"
)

// Mock parser functions (conceptual)
var (
	origExtractAuthorUsername  func(postHTMLBlock *goquery.Document) (string, error)
	origExtractTimestamp       func(postHTMLBlock *goquery.Document) (string, error)
	origExtractPostID          func(postHTMLBlock *goquery.Document) (string, error)
	origExtractPostOrderOnPage func(postHTMLBlock *goquery.Document) (int, error)
)

const mockPostHTMLForExtractorTest = `<html><body><p>Some post content</p></body></html>`

func TestExtractPostMetadata(t *testing.T) {
	tests := []struct {
		name                 string
		filePath             string
		mockAuthorUsername   string
		mockAuthorErr        error
		mockTimestamp        string
		mockTimestampErr     error
		mockPostID           string
		mockPostIDErr        error
		mockPostOrder        int
		mockPostOrderErr     error
		wantMetadata         data.PostMetadata
		wantErr              bool
		numExpectedErrs      int
		substringsInFinalErr []string
	}{
		{
			name:               "Valid full extraction",
			filePath:           "/archive/66/19618/page_1.html",
			mockAuthorUsername: "TestAuthor",
			mockTimestamp:      "2024-03-15 10:30:00",
			mockPostID:         "12345",
			mockPostOrder:      0,
			wantMetadata: data.PostMetadata{
				SubForumID:      "66",
				TopicID:         "19618",
				PageNumber:      1,
				AuthorUsername:  "TestAuthor",
				Timestamp:       "2024-03-15 10:30:00",
				PostID:          "12345",
				PostOrderOnPage: 0,
			},
			wantErr: false,
		},
		{
			name:     "Invalid file path - too short",
			filePath: "/page_1.html",
			wantMetadata: data.PostMetadata{
				SubForumID: "", TopicID: "", PageNumber: 0,
			},
			wantErr:              true,
			numExpectedErrs:      1,
			substringsInFinalErr: []string{"filePath '/page_1.html' is too short"},
		},
		{
			name:     "Invalid file path - bad page format",
			filePath: "/archive/66/19618/invalidpage.html",
			wantMetadata: data.PostMetadata{
				SubForumID: "66", TopicID: "19618", PageNumber: 0,
			},
			wantErr:              true,
			numExpectedErrs:      1,
			substringsInFinalErr: []string{"page filename 'invalidpage.html' does not match expected format"},
		},
		{
			name:             "Multiple extraction errors",
			filePath:         "/archive/77/20000/page_2.html",
			mockAuthorErr:    errors.New("author fail"),
			mockTimestampErr: errors.New("timestamp fail"),
			mockPostIDErr:    errors.New("postID fail"),
			mockPostOrderErr: errors.New("postOrder fail"),
			wantMetadata: data.PostMetadata{
				SubForumID:      "77",
				TopicID:         "20000",
				PageNumber:      2,
				AuthorUsername:  "",
				Timestamp:       "",
				PostID:          "",
				PostOrderOnPage: 0,
			},
			wantErr:         true,
			numExpectedErrs: 4,
			substringsInFinalErr: []string{
				"failed to extract author username: author fail",
				"failed to extract timestamp: timestamp fail",
				"failed to extract post ID: postID fail",
				"failed to extract post order on page: postOrder fail",
			},
		},
	}

	doc, _ := goquery.NewDocumentFromReader(strings.NewReader(mockPostHTMLForExtractorTest))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := ExtractPostMetadata(doc, tt.filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPostMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if metadata.SubForumID != tt.wantMetadata.SubForumID {
				t.Errorf("ExtractPostMetadata() metadata.SubForumID = %v, want %v", metadata.SubForumID, tt.wantMetadata.SubForumID)
			}
			if metadata.TopicID != tt.wantMetadata.TopicID {
				t.Errorf("ExtractPostMetadata() metadata.TopicID = %v, want %v", metadata.TopicID, tt.wantMetadata.TopicID)
			}
			if metadata.PageNumber != tt.wantMetadata.PageNumber {
				t.Errorf("ExtractPostMetadata() metadata.PageNumber = %v, want %v", metadata.PageNumber, tt.wantMetadata.PageNumber)
			}

			if tt.wantErr && err != nil {
				if tt.numExpectedErrs > 0 {
					expectedErrCountString := fmt.Sprintf("encountered %d error(s)", tt.numExpectedErrs)
					if tt.numExpectedErrs == 1 && !strings.Contains(err.Error(), "encountered") {
					} else if !strings.Contains(err.Error(), expectedErrCountString) {
						t.Errorf("ExtractPostMetadata() error string '%s' does not indicate expected %d errors with string '%s'", err.Error(), tt.numExpectedErrs, expectedErrCountString)
					}
				}
				for _, sub := range tt.substringsInFinalErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("ExtractPostMetadata() error string '%s' does not contain expected substring '%s'", err.Error(), sub)
					}
				}
			} else if !tt.wantErr {
				if metadata.AuthorUsername != tt.wantMetadata.AuthorUsername {
					t.Errorf("ExtractPostMetadata() metadata.AuthorUsername = %v, want %v", metadata.AuthorUsername, tt.wantMetadata.AuthorUsername)
				}
				if metadata.Timestamp != tt.wantMetadata.Timestamp {
					t.Errorf("ExtractPostMetadata() metadata.Timestamp = %v, want %v", metadata.Timestamp, tt.wantMetadata.Timestamp)
				}
				if metadata.PostID != tt.wantMetadata.PostID {
					t.Errorf("ExtractPostMetadata() metadata.PostID = %v, want %v", metadata.PostID, tt.wantMetadata.PostID)
				}
				if metadata.PostOrderOnPage != tt.wantMetadata.PostOrderOnPage {
					t.Errorf("ExtractPostMetadata() metadata.PostOrderOnPage = %v, want %v", metadata.PostOrderOnPage, tt.wantMetadata.PostOrderOnPage)
				}
			}
		})
	}
}
