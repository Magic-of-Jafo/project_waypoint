package parser

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// Utility to wrap a fragment in valid HTML for goquery
func wrapHTML(body string) *goquery.Document {
	fullHTML := `<!DOCTYPE html><html><body><table>` + body + `</table></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(fullHTML))
	if err != nil {
		panic("failed to parse test HTML: " + err.Error())
	}
	return doc
}

var (
	mockPostDoc1 = wrapHTML(`
<tr>
	<td class="normal bgc1 c w13 vat">
		<strong>TestUser1</strong><br />
		<a href="bb_profile.php?mode=view&amp;user=1"><img class="nb" src="images/avatars/nopic.gif" /></a><br />
		<span class="smalltext">
			<strong>Regular user</strong><br />SomeLocation<br />100 Posts
		</span><br />
		<a href="bb_profile.php?mode=view&amp;user=1"><img class="nb vab" src="images/profile.gif" /></a>
	</td>
	<td class="normal bgc1 vat w90">
		<div class="vt1 liketext">
			<div class="like_left">
				<a name="0"></a> <span class="b"><img class="nb vam" src="images/posticon.gif" />
				Posted: Mar 15, 2024 10:30 am</span>
			</div>
			<div class="like_right"><img class="vab" src="images/likes.gif" /><span id="p_12345">0</span></div>
		</div>
		<div class="w100">Post content here.</div>
	</td>
</tr>`)

	mockPostDocNoUsername = wrapHTML(`
<tr>
	<td class="normal bgc1 c w13 vat">
		<!-- No strong tag -->
	</td>
	<td class="normal bgc1 vat w90">
		<div class="vt1 liketext">
			<div class="like_left">
				<span class="b"><a name="0"></a><img src="images/posticon.gif" />
				Posted: Mar 15, 2024 10:30 am</span>
			</div>
			<div class="like_right"><span id="p_12345">0</span></div>
		</div>
	</td>
</tr>`)

	mockPostDocNoTimestamp = wrapHTML(`
<tr>
	<td class="normal bgc1 c w13 vat">
		<strong>TestUser1</strong>
	</td>
	<td class="normal bgc1 vat w90">
		<div class="vt1 liketext">
			<div class="like_left">
				<!-- Missing timestamp span -->
			</div>
			<div class="like_right"><span id="p_12345">0</span></div>
		</div>
	</td>
</tr>`)

	mockPostDocMalformedTimestamp = wrapHTML(`
<tr>
	<td class="normal bgc1 c w13 vat">
		<strong>TestUser1</strong>
	</td>
	<td class="normal bgc1 vat w90">
		<div class="vt1 liketext">
			<div class="like_left">
				<span class="b">Posted: March 15th, 2024, Bad Time</span>
			</div>
			<div class="like_right"><span id="p_12345">0</span></div>
		</div>
	</td>
</tr>`)

	mockPostDocNoPostID = wrapHTML(`
<tr>
	<td class="normal bgc1 c w13 vat">
		<strong>TestUser1</strong>
	</td>
	<td class="normal bgc1 vat w90">
		<div class="vt1 liketext">
			<div class="like_left">
				<span class="b">Posted: Mar 15, 2024 10:30 am</span>
			</div>
			<div class="like_right"><!-- No span with id="p_*" --></div>
		</div>
	</td>
</tr>`)

	mockPostDocNoPostOrder = wrapHTML(`
<tr>
	<td class="normal bgc1 c w13 vat">
		<strong>TestUser1</strong>
	</td>
	<td class="normal bgc1 vat w90">
		<div class="vt1 liketext">
			<div class="like_left">
				<span class="b">
					<!-- Missing anchor with name attribute -->
					Posted: Mar 15, 2024 10:30 am
				</span>
			</div>
			<div class="like_right"><span id="p_12345">0</span></div>
		</div>
	</td>
</tr>`)
)

func TestExtractAuthorUsername(t *testing.T) {
	tests := []struct {
		name     string
		doc      *goquery.Document
		wantUser string
		wantErr  bool
	}{
		{
			name:     "Valid username",
			doc:      mockPostDoc1,
			wantUser: "TestUser1",
			wantErr:  false,
		},
		{
			name:     "No username element",
			doc:      mockPostDocNoUsername,
			wantUser: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUser, err := ExtractAuthorUsername(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractAuthorUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotUser != tt.wantUser {
				t.Errorf("ExtractAuthorUsername() = %v, want %v", gotUser, tt.wantUser)
			}
		})
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name           string
		doc            *goquery.Document
		wantTimestamp  string
		wantErr        bool
		substringInErr string // If wantErr is true, check if error contains this
	}{
		{
			name:          "Valid timestamp",
			doc:           mockPostDoc1,
			wantTimestamp: "2024-03-15 10:30:00",
			wantErr:       false,
		},
		{
			name:           "No timestamp element",
			doc:            mockPostDocNoTimestamp,
			wantTimestamp:  "",
			wantErr:        true,
			substringInErr: "timestamp span.b not found with selector",
		},
		{
			name:           "Malformed timestamp string",
			doc:            mockPostDocMalformedTimestamp,
			wantTimestamp:  "",
			wantErr:        true,
			substringInErr: "failed to parse timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timestamp, err := ExtractTimestamp(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTimestamp() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if timestamp != tt.wantTimestamp {
				t.Errorf("ExtractTimestamp() gotTimestamp = %v, want %v", timestamp, tt.wantTimestamp)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.substringInErr) {
				t.Errorf("ExtractTimestamp() error string '%s' does not contain expected substring '%s'", err.Error(), tt.substringInErr)
			}
		})
	}
}

func TestExtractPostID(t *testing.T) {
	tests := []struct {
		name       string
		doc        *goquery.Document
		wantPostID string
		wantErr    bool
	}{
		{
			name:       "Valid PostID",
			doc:        mockPostDoc1,
			wantPostID: "12345",
			wantErr:    false,
		},
		{
			name:       "No PostID element",
			doc:        mockPostDocNoPostID,
			wantPostID: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postID, err := ExtractPostID(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPostID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if postID != tt.wantPostID {
				t.Errorf("ExtractPostID() gotPostID = %v, want %v", postID, tt.wantPostID)
			}
		})
	}
}

func TestExtractPostOrderOnPage(t *testing.T) {
	tests := []struct {
		name           string
		doc            *goquery.Document
		wantPostOrder  int
		wantErr        bool
		substringInErr string
	}{
		{
			name:          "Valid PostOrder",
			doc:           mockPostDoc1,
			wantPostOrder: 0,
			wantErr:       false,
		},
		{
			name:           "No PostOrder anchor element",
			doc:            mockPostDocNoPostOrder,
			wantPostOrder:  0,
			wantErr:        true,
			substringInErr: "post order anchor element not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			postOrder, err := ExtractPostOrderOnPage(tt.doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractPostOrderOnPage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if postOrder != tt.wantPostOrder {
				t.Errorf("ExtractPostOrderOnPage() gotPostOrder = %v, want %v", postOrder, tt.wantPostOrder)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.substringInErr) {
				t.Errorf("ExtractPostOrderOnPage() error string '%s' does not contain expected substring '%s'", err.Error(), tt.substringInErr)
			}
		})
	}
}
