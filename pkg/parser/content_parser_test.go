package parser

import (
	"strings"
	"testing"

	"project-waypoint/pkg/data" // Added for ContentBlock types

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)

func TestExtractQuoteDetails(t *testing.T) {
	tests := []struct {
		name              string
		htmlInput         string
		expectedUser      string
		expectedTimestamp string // Raw extracted, not parsed to YYYY-MM-DD HH:MM:SS for this test yet
		expectedText      string
		expectError       bool
	}{
		{
			name: "Simple quote with username wrote:",
			htmlInput: `<table class="cfq">
					<tr><td><b>User1 wrote:</b><br/>Jan 1, 2023, 10:00 AM</td></tr>
					<tr><td>This is the quote text.</td></tr>
				</table>`,
			expectedUser:      "User1",
			expectedTimestamp: "Jan 1, 2023, 10:00 AM",
			expectedText:      "This is the quote text.",
			expectError:       false,
		},
		{
			name: "Quote with Quote: Username format",
			htmlInput: `<table class="cfq">
					<tr><td><b>Quote: User2</b><br/>Feb 10, 2023, 11:30 PM</td></tr>
					<tr><td>Another quote.</td></tr>
				</table>`,
			expectedUser:      "User2",
			expectedTimestamp: "Feb 10, 2023, 11:30 PM",
			expectedText:      "Another quote.",
			expectError:       false,
		},
		{
			name: "Quote with username only",
			htmlInput: `<table class="cfq">
					<tr><td><b>User3</b><br/>Mar 15, 2023, 09:15 AM</td></tr>
					<tr><td>Text here.</td></tr>
				</table>`,
			expectedUser:      "User3",
			expectedTimestamp: "Mar 15, 2023, 09:15 AM",
			expectedText:      "Text here.",
			expectError:       false,
		},
		{
			name: "Quote no timestamp",
			htmlInput: `<table class="cfq">
					<tr><td><b>User4 wrote:</b></td></tr>
					<tr><td>Quote without a timestamp.</td></tr>
				</table>`,
			expectedUser:      "User4",
			expectedTimestamp: "",
			expectedText:      "Quote without a timestamp.",
			expectError:       false,
		},
		{
			name: "Quote with text including HTML",
			htmlInput: `<table class="cfq">
					<tr><td><b>User5 wrote:</b><br/>Apr 1, 2023, 12:00 PM</td></tr>
					<tr><td>This is <i>italicized</i> and <b>bold</b>.</td></tr>
				</table>`,
			expectedUser:      "User5",
			expectedTimestamp: "Apr 1, 2023, 12:00 PM",
			expectedText:      "This is <i>italicized</i> and <b>bold</b>.",
			expectError:       false,
		},
		{
			name: "Malformed quote - no attribution cell with <b>",
			htmlInput: `<table class="cfq">
					<tr><td>No b tag here</td></tr>
					<tr><td>Some quote text.</td></tr>
				</table>`,
			expectError: true,
		},
		{
			name: "Malformed quote - no text cell (though current logic might find first td as text if no attr)",
			htmlInput: `<table class="cfq">
					<tr><td><b>User6 wrote:</b></td></tr>
				</table>`,
			expectedUser:      "User6",
			expectedTimestamp: "",
			expectedText:      "",    // Expect empty as no clear text cell found by current logic
			expectError:       false, // Not an error, but empty text
		},
		// TODO: Add more tests: variations in td structure, nested tables (if that's a concern for cfq text part)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.htmlInput))
			assert.NoError(t, err, "Error creating document from reader for test: %s", tt.name)

			quoteElement := doc.Find("table.cfq").First()
			assert.NotNil(t, quoteElement, "quoteElement should not be nil for test: %s", tt.name)

			user, timestamp, text, err := ExtractQuoteDetails(quoteElement)

			if tt.expectError {
				assert.Error(t, err, "Expected an error for test: %s", tt.name)
			} else {
				assert.NoError(t, err, "Did not expect an error for test: %s", tt.name)
				assert.Equal(t, tt.expectedUser, user, "User mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedTimestamp, timestamp, "Timestamp mismatch for test: %s", tt.name)
				assert.Equal(t, tt.expectedText, text, "Text mismatch for test: %s", tt.name)
			}
		})
	}
}

func TestParseContentBlocks(t *testing.T) {
	tests := []struct {
		name           string
		htmlInput      string // This will be HTML for the *post block* (e.g., the content of td.normal.bgc1.vat.w90)
		expectedBlocks []data.ContentBlock
		expectError    bool
	}{
		{
			name: "Only new_text",
			htmlInput: `<div class="w100">
					This is some new text. <br>With a line break.
				</div>`,
			expectedBlocks: []data.ContentBlock{
				{Type: data.ContentBlockTypeNewText, Content: "This is some new text. <br/>With a line break."},
			},
			expectError: false,
		},
		{
			name: "Only a quote block",
			htmlInput: `<div class="w100">
					<table class="cfq">
						<tr><td><b>User1 wrote:</b><br/>Jan 1, 2023, 10:00 AM</td></tr>
						<tr><td>This is the quote text.</td></tr>
					</table>
				</div>`,
			expectedBlocks: []data.ContentBlock{
				{Type: data.ContentBlockTypeQuote, QuotedUser: "User1", QuotedTimestamp: "Jan 1, 2023, 10:00 AM", QuotedText: "This is the quote text."},
			},
			expectError: false,
		},
		{
			name: "New text then quote",
			htmlInput: `<div class="w100">
					Author text first.
					<table class="cfq">
						<tr><td><b>QUser wrote:</b></td></tr>
						<tr><td>Quoted part.</td></tr>
					</table>
				</div>`,
			expectedBlocks: []data.ContentBlock{
				{Type: data.ContentBlockTypeNewText, Content: "Author text first."},
				{Type: data.ContentBlockTypeQuote, QuotedUser: "QUser", QuotedTimestamp: "", QuotedText: "Quoted part."},
			},
			expectError: false,
		},
		{
			name: "Quote then new text",
			htmlInput: `<div class="w100">
					<table class="cfq">
						<tr><td><b>QUser2 wrote:</b></td></tr>
						<tr><td>Another quoted part.</td></tr>
					</table>
					Followed by author text.
				</div>`,
			expectedBlocks: []data.ContentBlock{
				{Type: data.ContentBlockTypeQuote, QuotedUser: "QUser2", QuotedTimestamp: "", QuotedText: "Another quoted part."},
				{Type: data.ContentBlockTypeNewText, Content: "Followed by author text."},
			},
			expectError: false,
		},
		{
			name: "Mixed new text and multiple quotes",
			htmlInput: `<div class="w100">
					Text 1
					<table class="cfq">
						<tr><td><b>UserA wrote:</b></td></tr><tr><td>Quote A</td></tr>
					</table>
					Text 2 <a href="#">link</a>
					<table class="cfq">
						<tr><td><b>UserB wrote:</b></td></tr><tr><td>Quote B</td></tr>
					</table>
					Text 3
				</div>`,
			expectedBlocks: []data.ContentBlock{
				{Type: data.ContentBlockTypeNewText, Content: "Text 1"},
				{Type: data.ContentBlockTypeQuote, QuotedUser: "UserA", QuotedText: "Quote A"},
				{Type: data.ContentBlockTypeNewText, Content: "Text 2 <a href=\"#\">link</a>"},
				{Type: data.ContentBlockTypeQuote, QuotedUser: "UserB", QuotedText: "Quote B"},
				{Type: data.ContentBlockTypeNewText, Content: "Text 3"},
			},
			expectError: false,
		},
		{
			name:           "Empty content container div",
			htmlInput:      `<div class="w100"></div>`,
			expectedBlocks: []data.ContentBlock{},
			expectError:    false,
		},
		{
			name:      "No content container div (should be handled by caller, but ParseContentBlocks might return empty)",
			htmlInput: `<div>Some other content</div>`,
			expectedBlocks: []data.ContentBlock{
				{Type: data.ContentBlockTypeNewText, Content: "Some other content"},
			},
			expectError: false, // Current logic returns empty, logs warning
		},
		// TODO: Add more complex cases: nested non-cfq tables, weird spacing, html comments
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.htmlInput))
			assert.NoError(t, err, "Error creating document from reader for test: %s", tt.name)

			// The tt.htmlInput is assumed to be the content container itself.
			// goquery.NewDocumentFromReader wraps input in <html><body> if not present.
			// So, we find the first child of body, which should be our container.
			// If htmlInput is empty or just whitespace, Children() might be empty.
			var containerToTest *goquery.Selection
			if strings.TrimSpace(tt.htmlInput) == "" {
				// Create an empty selection if input is empty, to match behavior of Find if nothing found.
				containerToTest = doc.Find("this_will_not_match_anything_hopefully")
			} else {
				containerToTest = doc.Find("body").Children().First()
			}

			blocks, err := ParseContentBlocks(containerToTest)

			if tt.expectError {
				assert.Error(t, err, "Expected an error for test: %s", tt.name)
			} else {
				assert.NoError(t, err, "Did not expect an error for test: %s", tt.name)
			}

			// Further assertions only if no error was expected or if an error occurred but we still want to check blocks (e.g. partial parse)
			if !tt.expectError || (tt.expectError && err != nil) { // Adjust condition as needed for your error handling tests
				assert.Equal(t, len(tt.expectedBlocks), len(blocks), "Number of blocks mismatch for test: %s", tt.name)
				if len(tt.expectedBlocks) == len(blocks) { // Only compare content if counts match
					for i, expectedBlock := range tt.expectedBlocks {
						assert.Equal(t, expectedBlock.Type, blocks[i].Type, "Block type mismatch at index %d for test: %s", i, tt.name)
						assert.Equal(t, expectedBlock.Content, blocks[i].Content, "Block content mismatch at index %d for test: %s", i, tt.name)
						assert.Equal(t, expectedBlock.QuotedUser, blocks[i].QuotedUser, "Block quoted user mismatch at index %d for test: %s", i, tt.name)
						assert.Equal(t, expectedBlock.QuotedTimestamp, blocks[i].QuotedTimestamp, "Block quoted timestamp mismatch at index %d for test: %s", i, tt.name)
						assert.Equal(t, expectedBlock.QuotedText, blocks[i].QuotedText, "Block quoted text mismatch at index %d for test: %s", i, tt.name)
					}
				}
			}
		})
	}
}

// TODO: Add tests for ParseContentBlocks
// Test cases should include:
// - Only new_text
// - Only a quote block
// - new_text then quote
// - quote then new_text
// - multiple new_text and quote blocks, in various orders
// - empty input
// - input that doesn't match the expected post content container structure
// - posts with <br> tags in new_text
// - posts with links or other inline HTML in new_text
