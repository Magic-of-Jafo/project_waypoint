package parser

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

const (
	mockPostHTML1 = `
<tr>
    <td class="normal bgc1 c w13 vat">
        <strong>TestUser1</strong><br />
        <a href="bb_profile.php?mode=view&amp;user=1"><img class="nb" src="images/avatars/nopic.gif" vspace="3" alt="View Profile" title="View Profile" /></a><br /><span class="smalltext">
        <strong>Regular user</strong><br />
        SomeLocation<br />
        100 Posts</span><br />
        <a href="bb_profile.php?mode=view&amp;user=1"><img class="nb vab" src="images/profile.gif" alt="Profile of TestUser1" title="Profile of TestUser1" /></a>
    </td>
    <td class="normal bgc1 vat w90">
        <div class="vt1 liketext">
            <div class="like_left">
                <span class="b">
                <a name="0"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
                Posted: Mar 15, 2024 10:30 am </span>&nbsp;&nbsp;
            </div>
            <div class="like_right"><img class="vab" src="images/likes.gif" alt="There are no likes for this post." title="There are no  likes for this post." /><span id="p_12345">0</span></div>
        </div>
        <div class="w100">
            Post content here.
        </div>
    </td>
</tr>`

	mockPostHTMLNoUsername = `
<tr>
    <td class="normal bgc1 c w13 vat">
        <!-- No username strong tag -->
    </td>
    <td class="normal bgc1 vat w90">
        <div class="vt1 liketext">
            <div class="like_left">
                <span class="b">
                <a name="0"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
                Posted: Mar 15, 2024 10:30 am </span>&nbsp;&nbsp;
            </div>
            <div class="like_right"><span id="p_12345">0</span></div>
        </div>
    </td>
</tr>`

	mockPostHTMLNoTimestamp = `
<tr>
    <td class="normal bgc1 c w13 vat">
        <strong>TestUser1</strong>
    </td>
    <td class="normal bgc1 vat w90">
        <div class="vt1 liketext">
            <div class="like_left">
                <!-- No timestamp span -->
            </div>
            <div class="like_right"><span id="p_12345">0</span></div>
        </div>
    </td>
</tr>`
	mockPostHTMLMalformedTimestamp = `
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
</tr>`
	mockPostHTMLNoPostID = `
<tr>
    <td class="normal bgc1 c w13 vat">
        <strong>TestUser1</strong>
    </td>
    <td class="normal bgc1 vat w90">
        <div class="vt1 liketext">
            <div class="like_left">
                <span class="b">
                <a name="0"></a><img class="nb vam" src="images/posticon.gif" alt="Post Icon" title="Post Icon" /> 
                Posted: Mar 15, 2024 10:30 am </span>&nbsp;&nbsp;
            </div>
            <div class="like_right"><!-- No Post ID span --></div>
        </div>
    </td>
</tr>`
	mockPostHTMLNoPostOrder = `
<tr>
    <td class="normal bgc1 c w13 vat">
        <strong>TestUser1</strong>
    </td>
    <td class="normal bgc1 vat w90">
        <div class="vt1 liketext">
            <div class="like_left">
                <span class="b">
                <!-- No anchor for post order -->
                Posted: Mar 15, 2024 10:30 am </span>&nbsp;&nbsp;
            </div>
            <div class="like_right"><span id="p_12345">0</span></div>
        </div>
    </td>
</tr>`
)

func TestExtractAuthorUsername(t *testing.T) {
	tests := []struct {
		name        string
		htmlContent string
		wantUser    string
		wantErr     bool
	}{
		{
			name:        "Valid username",
			htmlContent: mockPostHTML1,
			wantUser:    "TestUser1",
			wantErr:     false,
		},
		{
			name:        "No username element",
			htmlContent: mockPostHTMLNoUsername,
			wantUser:    "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.htmlContent))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}
			user, err := ExtractAuthorUsername(doc)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractAuthorUsername() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if user != tt.wantUser {
				t.Errorf("ExtractAuthorUsername() gotUser = %v, want %v", user, tt.wantUser)
			}
		})
	}
}

func TestExtractTimestamp(t *testing.T) {
	tests := []struct {
		name           string
		htmlContent    string
		wantTimestamp  string
		wantErr        bool
		substringInErr string // If wantErr is true, check if error contains this
	}{
		{
			name:          "Valid timestamp",
			htmlContent:   mockPostHTML1,
			wantTimestamp: "2024-03-15 10:30:00",
			wantErr:       false,
		},
		{
			name:           "No timestamp element",
			htmlContent:    mockPostHTMLNoTimestamp,
			wantTimestamp:  "",
			wantErr:        true,
			substringInErr: "timestamp element not found",
		},
		{
			name:           "Malformed timestamp string",
			htmlContent:    mockPostHTMLMalformedTimestamp,
			wantTimestamp:  "",
			wantErr:        true,
			substringInErr: "failed to parse timestamp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.htmlContent))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}
			timestamp, err := ExtractTimestamp(doc)
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
		name        string
		htmlContent string
		wantPostID  string
		wantErr     bool
	}{
		{
			name:        "Valid PostID",
			htmlContent: mockPostHTML1,
			wantPostID:  "12345",
			wantErr:     false,
		},
		{
			name:        "No PostID element",
			htmlContent: mockPostHTMLNoPostID,
			wantPostID:  "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.htmlContent))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}
			postID, err := ExtractPostID(doc)
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
		htmlContent    string
		wantPostOrder  int
		wantErr        bool
		substringInErr string
	}{
		{
			name:          "Valid PostOrder",
			htmlContent:   mockPostHTML1,
			wantPostOrder: 0,
			wantErr:       false,
		},
		{
			name:           "No PostOrder anchor element",
			htmlContent:    mockPostHTMLNoPostOrder,
			wantPostOrder:  0, // Expect 0 for error cases returning int
			wantErr:        true,
			substringInErr: "post order anchor element not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := goquery.NewDocumentFromReader(strings.NewReader(tt.htmlContent))
			if err != nil {
				t.Fatalf("Failed to parse HTML: %v", err)
			}
			postOrder, err := ExtractPostOrderOnPage(doc)
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
