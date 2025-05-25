package navigation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestFetchHTML(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "<html><body><h1>Hello, World!</h1></body></html>")
		}))
		defer server.Close()

		html, err := FetchHTML(server.URL)
		if err != nil {
			t.Errorf("FetchHTML() error = %v, wantErr %v", err, false)
		}

		expectedHTML := "<html><body><h1>Hello, World!</h1></body></html>\n"
		if html != expectedHTML {
			t.Errorf("FetchHTML() html = %q, want %q", html, expectedHTML)
		}
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := FetchHTML(server.URL)
		if err == nil {
			t.Errorf("FetchHTML() error = %v, wantErr %v", err, true)
		}
	})

	t.Run("invalid URL", func(t *testing.T) {
		_, err := FetchHTML("invalid-url")
		if err == nil {
			t.Errorf("FetchHTML() error = %v, wantErr %v", err, true)
		}
	})
}

func TestParsePaginationLinks(t *testing.T) {
	const forumBaseURL = "https://www.themagiccafe.com/forums/viewforum.php"
	const topicsPerPage = 30

	tests := []struct {
		name           string
		pageURL        string
		htmlContent    string
		wantLinks      []string
		wantErr        bool
		isErrSubstring string
	}{
		{
			name:    "multiple pages",
			pageURL: fmt.Sprintf("%s?forum=54", forumBaseURL),
			htmlContent: `
				<table class="normal" cellpadding="4" cellspacing="1">
					<tr>
						<td class="normal bgc1 b midtext" colspan="6">
							&nbsp;Go to page <span class="on_page">1</span>~
							<a href="viewforum.php?forum=54&amp;start=30">2</a>~
							<a href="viewforum.php?forum=54&amp;start=60">3</a>
							[<a href="viewforum.php?forum=54&amp;start=30">Next</a>]
						</td>
					</tr>
				</table>`,
			wantLinks: []string{
				fmt.Sprintf("%s?forum=54", forumBaseURL),
				fmt.Sprintf("%s?forum=54&start=30", forumBaseURL),
				fmt.Sprintf("%s?forum=54&start=60", forumBaseURL),
			},
			wantErr: false,
		},
		{
			name:    "single page",
			pageURL: fmt.Sprintf("%s?forum=100", forumBaseURL),
			htmlContent: `
				<table class="normal" cellpadding="4" cellspacing="1">
					<tr><td class="normal bgc1 b midtext" colspan="6">&nbsp;Go to page <span class="on_page">1</span></td></tr>
				</table>`,
			wantLinks: []string{
				fmt.Sprintf("%s?forum=100", forumBaseURL),
			},
			wantErr: false,
		},
		{
			name:    "realistic magic cafe first page",
			pageURL: "https://www.themagiccafe.com/forums/viewforum.php?forum=54",
			htmlContent: `
				<body bgcolor="#000000"><div id="container">
				<table class="normal" cellpadding="4" cellspacing="1">
					<tr class="normal bgc1">
						<td class="mltext" colspan="6">
							<table class="w100"><tr><td class="w75">Index</td></tr></table>
						</td>
					</tr>
					<tr>
						<td class="normal bgc1 b midtext" colspan="6">
							&nbsp;Go to page <span class="on_page">1</span>~
							<a href="viewforum.php?forum=54&amp;start=30" title="Page 2" alt="Page 2">2</a>~
							<a href="viewforum.php?forum=54&amp;start=60" title="Page 3" alt="Page 3">3</a>..
							<a href="viewforum.php?forum=54&amp;start=1890" title="Page 64" alt="Page 64">64</a> 
							[<a href="viewforum.php?forum=54&amp;start=30" title="Next Page" alt="Next Page">Next</a>]
						</td>
					</tr>
				</table></div></body>`,
			wantLinks: func() []string {
				var links []string
				links = append(links, "https://www.themagiccafe.com/forums/viewforum.php?forum=54")
				for i := 1; i < 64; i++ {
					links = append(links, fmt.Sprintf("https://www.themagiccafe.com/forums/viewforum.php?forum=54&start=%d", i*topicsPerPage))
				}
				return links
			}(),
			wantErr: false,
		},
		{
			name:           "invalid page URL",
			pageURL:        "://invalid-url",
			htmlContent:    ``,
			wantLinks:      nil,
			wantErr:        true,
			isErrSubstring: "failed to parse pageURL",
		},
		{
			name:           "missing forum ID in page URL",
			pageURL:        forumBaseURL, // No query params
			htmlContent:    `<td class="normal bgc1 b midtext"><a href="viewforum.php?forum=54&amp;start=30">2</a></td>`,
			wantLinks:      nil,
			wantErr:        true,
			isErrSubstring: "forum ID not found",
		},
		{
			name:        "no pagination links found",
			pageURL:     fmt.Sprintf("%s?forum=54", forumBaseURL),
			htmlContent: `<body><p>No links here</p></body>`,
			wantLinks:   []string{fmt.Sprintf("%s?forum=54", forumBaseURL)}, // Should still return current page
			wantErr:     false,
		},
		{
			name:        "malformed href in pagination",
			pageURL:     fmt.Sprintf("%s?forum=54", forumBaseURL),
			htmlContent: `<td class="normal bgc1 b midtext"><a href="://malformed">2</a></td>`,
			wantLinks:   []string{fmt.Sprintf("%s?forum=54", forumBaseURL)}, // Expect current page, malformed link ignored
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLinks, err := ParsePaginationLinks(tt.htmlContent, tt.pageURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePaginationLinks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.isErrSubstring != "" {
				if !strings.Contains(err.Error(), tt.isErrSubstring) {
					t.Errorf("ParsePaginationLinks() error = %v, want error containing %q", err, tt.isErrSubstring)
				}
			}
			// Sort both slices for consistent comparison
			sort.Strings(gotLinks)
			sort.Strings(tt.wantLinks)
			if !reflect.DeepEqual(gotLinks, tt.wantLinks) {
				t.Errorf("ParsePaginationLinks() gotLinks = %v, want %v", gotLinks, tt.wantLinks)
			}
		})
	}
}

// TODO: Add unit tests for navigation logic here
