package extractorlogic

import (
	"strconv"
	"strings"
	"testing"

	"project-waypoint/pkg/data"

	"github.com/PuerkitoBio/goquery"
)

// Utility to wrap a fragment in valid HTML for goquery
// Copied from pkg/parser/parser_test.go
func wrapHTML(body string) *goquery.Document {
	fullHTML := `<!DOCTYPE html><html><body><table>` + body + `</table></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(fullHTML))
	if err != nil {
		panic("failed to parse test HTML: " + err.Error())
	}
	return doc
}

// var (
// 	_ func(postHTMLBlock *goquery.Document) (string, error)
// 	_ func(postHTMLBlock *goquery.Document) (string, error)
// 	_ func(postHTMLBlock *goquery.Document) (string, error)
// 	_ func(postHTMLBlock *goquery.Document) (int, error)
// )

const mockPostHTMLFragmentForExtractorTest = `
<tr>
    <td class="normal bgc1 c w13 vat">
        <strong>TestAuthor</strong>
    </td>
    <td class="normal bgc1 vat w90">
        <div class="vt1 liketext">
            <div class="like_left">
                <span class="b">Posted: Jan 1, 2023 01:02 pm</span>
                <a name="0"></a>
            </div>
            <div class="like_right">
                <span id="p_12345"></span>
            </div>
        </div>
    </td>
</tr>
`

func TestExtractPostMetadata(t *testing.T) {
	tests := []struct {
		name                 string
		filePath             string
		wantMetadata         data.PostMetadata
		wantErr              bool
		numExpectedErrs      int
		substringsInFinalErr []string
	}{
		{
			name:     "Valid full extraction",
			filePath: "/archive/66/19618/page_1.html",
			wantMetadata: data.PostMetadata{
				SubForumID:      "66",
				TopicID:         "19618",
				PageNumber:      1,
				AuthorUsername:  "TestAuthor",
				Timestamp:       "2023-01-01 13:02:00",
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
				AuthorUsername: "", Timestamp: "", PostID: "", PostOrderOnPage: 0,
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
				AuthorUsername: "", Timestamp: "", PostID: "", PostOrderOnPage: 0,
			},
			wantErr:              true,
			numExpectedErrs:      1,
			substringsInFinalErr: []string{"page filename 'invalidpage.html' does not match expected format"},
		},
		{
			name:     "Multiple errors - path error and (hypothetically) parser errors if HTML was bad",
			filePath: "/archive/ERROR/ERROR/page_ERROR.html",
			wantMetadata: data.PostMetadata{
				SubForumID:      "ERROR",
				TopicID:         "ERROR",
				PageNumber:      0,
				AuthorUsername:  "", Timestamp: "", PostID: "", PostOrderOnPage: 0,
			},
			wantErr: true,
			numExpectedErrs: 1,
			substringsInFinalErr: []string{
				"failed to parse page number from 'page_ERROR.html'",
			},
		},
	}

	doc := wrapHTML(mockPostHTMLFragmentForExtractorTest)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, err := ExtractPostMetadata(doc, tt.filePath)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPostMetadata() error = %v, wantErr %v", err, tt.wantErr)
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

			if !tt.wantErr {
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

			if tt.wantErr && err != nil {
				if tt.numExpectedErrs > 0 {
					actualErrCount := 0
					if strings.Contains(err.Error(), "encountered") && strings.Contains(err.Error(), "error(s) during metadata extraction:") {
						parts := strings.SplitN(err.Error(), ":", 2)
						if len(parts) > 1 {
							countStrPart := strings.Fields(parts[0])[1]
							parsedCount, e := strconv.Atoi(countStrPart)
							if e == nil {
								actualErrCount = parsedCount
							} else {
								t.Logf("Could not parse error count from: %s", parts[0])
								actualErrCount = 1
							}
						}
					} else if err.Error() != "" {
						actualErrCount = 1
					}

					if actualErrCount != tt.numExpectedErrs {
						t.Errorf("ExtractPostMetadata() error count = %d, want %d. Error: %v", actualErrCount, tt.numExpectedErrs, err)
					}
				}
				for _, sub := range tt.substringsInFinalErr {
					if !strings.Contains(err.Error(), sub) {
						t.Errorf("ExtractPostMetadata() error string '%s' does not contain expected substring '%s'", err.Error(), sub)
					}
				}
			} else if tt.wantErr && err == nil {
				t.Errorf("ExtractPostMetadata() expected an error, but got nil")
			}
		})
	}
}
