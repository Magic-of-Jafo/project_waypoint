package topic

import (
	"reflect"
	"testing"
)

func TestExtractTopics(t *testing.T) {
	const pageBaseURL = "https://www.themagiccafe.com/forums/viewforum.php?forum=54"

	tests := []struct {
		name        string
		htmlContent string
		pageURL     string
		wantTopics  []TopicInfo
		wantErr     bool
	}{
		{
			name:    "sticky and regular topics",
			pageURL: pageBaseURL,
			htmlContent: `
<table class="normal" cellpadding="4" cellspacing="1">
    <tr>
        <td class="normal bgc2 w5 c nowrap"><img src="images/folder.gif" /></td>
        <td class="normal bgc2">
            <img class="vam" src="images/sticky.gif" alt="This Topic is ''Sticky''" />&nbsp;<a class="b" href="viewtopic.php?topic=286725&forum=54">Silk Sizes Explained</a>
        </td>
    </tr>
    <tr>
        <td class="normal bgc2 w5 c nowrap"><img src="images/red_folder.gif" /></td>
        <td class="normal bgc2">
            <a class="b" href="viewtopic.php?topic=780955&forum=54">Purse Swindle by Alexander De Cova</a>
        </td>
    </tr>
</table>`,
			wantTopics: []TopicInfo{
				{ID: "286725", Title: "Silk Sizes Explained", URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic=286725&forum=54"},
				{ID: "780955", Title: "Purse Swindle by Alexander De Cova", URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic=780955&forum=54"},
			},
			wantErr: false,
		},
		{
			name:    "topic with no topic id in href",
			pageURL: pageBaseURL,
			htmlContent: `
<table class="normal">
    <tr>
        <td class="normal bgc2"><a class="b" href="viewtopic.php?forum=54">A Topic With No ID</a></td>
    </tr>
</table>`,
			wantTopics: []TopicInfo{},
			wantErr:    false, // Should log but not error, returns empty slice
		},
		{
			name:    "topic with empty title",
			pageURL: pageBaseURL,
			htmlContent: `
<table class="normal">
    <tr>
        <td class="normal bgc2"><a class="b" href="viewtopic.php?topic=123&forum=54"></a></td>
    </tr>
</table>`,
			wantTopics: []TopicInfo{},
			wantErr:    false, // Should skip and return empty slice
		},
		{
			name:        "no topics found",
			pageURL:     pageBaseURL,
			htmlContent: `<p>No topics here.</p>`,
			wantTopics:  []TopicInfo{},
			wantErr:     false,
		},
		{
			name:        "malformed html",
			pageURL:     pageBaseURL,
			htmlContent: `<<<<<>>>>>`,
			wantTopics:  nil, // goquery.NewDocumentFromReader will error
			wantErr:     true,
		},
		{
			name:    "duplicate topic ID on page",
			pageURL: pageBaseURL,
			htmlContent: `
<table class="normal">
    <tr>
        <td class="normal bgc2"><a class="b" href="viewtopic.php?topic=111&forum=54">Topic Alpha</a></td>
    </tr>
    <tr>
        <td class="normal bgc2"><a class="b" href="viewtopic.php?topic=111&forum=54">Topic Alpha Duplicate</a></td>
    </tr>
    <tr>
        <td class="normal bgc2"><a class="b" href="viewtopic.php?topic=222&forum=54">Topic Beta</a></td>
    </tr>
</table>`,
			wantTopics: []TopicInfo{
				{ID: "111", Title: "Topic Alpha", URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic=111&forum=54"},
				{ID: "222", Title: "Topic Beta", URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic=222&forum=54"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTopics, err := ExtractTopics(tt.htmlContent, tt.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExtractTopics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotTopics, tt.wantTopics) {
				t.Errorf("ExtractTopics() gotTopics = %v, want %v", gotTopics, tt.wantTopics)
			}
		})
	}
}

// Removed placeholder: // TODO: Add unit tests for topic extraction logic here
