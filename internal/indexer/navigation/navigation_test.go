package navigation

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"waypoint_archive_scripts/pkg/data" // Added for data.Topic
)

func TestFetchHTML(t *testing.T) {
	t.Run("successful fetch no delay - default impl", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "<html><body>Hello</body></html>")
		}))
		defer server.Close()
		html, err := defaultFetchHTML(server.URL, 0)
		if err != nil {
			t.Errorf("defaultFetchHTML() error = %v, wantErr false", err)
		}
		if !strings.Contains(html, "Hello") {
			t.Errorf("defaultFetchHTML() html = %q, want contains Hello", html)
		}
	})

	t.Run("successful fetch with delay - default impl", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "<html><body>Delayed Hello</body></html>")
		}))
		defer server.Close()
		startTime := time.Now()
		html, err := defaultFetchHTML(server.URL, 10*time.Millisecond)
		duration := time.Since(startTime)
		if err != nil {
			t.Errorf("defaultFetchHTML() error = %v, wantErr false", err)
		}
		if !strings.Contains(html, "Delayed Hello") {
			t.Errorf("defaultFetchHTML() html = %q, want contains Delayed Hello", html)
		}
		if duration < 10*time.Millisecond {
			t.Errorf("defaultFetchHTML() delay was too short, got %v, want >= %v", duration, 10*time.Millisecond)
		}
	})

	t.Run("server error - global FetchHTML", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		_, err := FetchHTML(server.URL, 0)
		if err == nil {
			t.Errorf("FetchHTML() error = %v, wantErr %v", err, true)
		}
	})

	t.Run("invalid URL - global FetchHTML", func(t *testing.T) {
		_, err := FetchHTML("invalid-url", 0)
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

func TestGetTopicPageURLs(t *testing.T) {
	tests := []struct {
		name         string
		topicDetails data.Topic
		mockHTML     map[string]string // Map URL to mock HTML content
		wantNavInfos []PageNavigationInfo
		wantErr      bool
	}{
		{
			name: "single page topic",
			topicDetails: data.Topic{
				ID:  "12345",
				URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic_id=12345",
			},
			mockHTML: map[string]string{
				"https://www.themagiccafe.com/forums/viewtopic.php?topic_id=12345": `<html><body><p>Single page content</p></body></html>`,
			},
			wantNavInfos: []PageNavigationInfo{
				{PageNumber: 1, URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic_id=12345"},
			},
			wantErr: false,
		},
		{
			name: "multi page topic",
			topicDetails: data.Topic{
				ID:  "67890",
				URL: "http://example.com/viewtopic.php?topic_id=67890",
			},
			mockHTML: map[string]string{
				"http://example.com/viewtopic.php?topic_id=67890":        `<html><body>Page 1 <a href="viewtopic.php?topic_id=67890&page=2">Next</a></body></html>`,
				"http://example.com/viewtopic.php?topic_id=67890&page=2": `<html><body>Page 2 <a href="viewtopic.php?topic_id=67890&page=3">Next</a></body></html>`,
				"http://example.com/viewtopic.php?topic_id=67890&page=3": `<html><body>Page 3 No Next Link</body></html>`,
			},
			wantNavInfos: []PageNavigationInfo{
				{PageNumber: 1, URL: "http://example.com/viewtopic.php?topic_id=67890"},
				{PageNumber: 2, URL: "http://example.com/viewtopic.php?topic_id=67890&page=2"},
				{PageNumber: 3, URL: "http://example.com/viewtopic.php?topic_id=67890&page=3"},
			},
			wantErr: false,
		},
		{
			name: "empty topic ID",
			topicDetails: data.Topic{
				ID:  "", // Empty ID
				URL: "https://www.themagiccafe.com/forums/viewtopic.php?topic_id=",
			},
			wantErr: true,
		},
		{
			name: "invalid topic URL",
			topicDetails: data.Topic{
				ID:  "123",
				URL: "://not-a-url",
			},
			wantErr: true,
		},
		{
			name: "fetch error on first page",
			topicDetails: data.Topic{
				ID:  "fetcherror1",
				URL: "http://example.com/fetcherror1",
			},
			mockHTML: map[string]string{ // Intentionally empty to cause fetch error with specific check
				// "http://example.com/fetcherror1": "", // This will be handled by error in mock FetchHTML
			},
			wantNavInfos: []PageNavigationInfo{ // Should return what it had, which is the first page attempt
				{PageNumber: 1, URL: "http://example.com/fetcherror1"},
			},
			wantErr: true, // Error from FetchHTML
		},
		{
			name: "fetch error on second page",
			topicDetails: data.Topic{
				ID:  "fetcherror2",
				URL: "http://example.com/fetcherror2_page1", // First page URL
			},
			mockHTML: map[string]string{
				"http://example.com/fetcherror2_page1": `<html><body>Page 1 <a href="fetcherror2_page2">Next</a></body></html>`,
				// No HTML needed for fetcherror2_page2 as it will cause a fetch error in the mock
			},
			wantNavInfos: []PageNavigationInfo{
				{PageNumber: 1, URL: "http://example.com/fetcherror2_page1"},
				{PageNumber: 2, URL: "http://example.com/fetcherror2_page2"}, // This URL is determined, then fetch fails
			},
			wantErr: true, // Error from FetchHTML on second page
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalFetchHTML := FetchHTML
			defer func() { FetchHTML = originalFetchHTML }()

			FetchHTML = func(fetchURL string, delay time.Duration) (string, error) {
				if tt.name == "fetch error on first page" && fetchURL == tt.topicDetails.URL {
					return "", fmt.Errorf("mock fetch error for first page")
				}
				expectedSecondPageURLForFetchError2 := ""
				if tt.topicDetails.ID == "fetcherror2" {
					parsedBase, errBase := url.Parse(tt.topicDetails.URL)
					if errBase != nil {
						return "", fmt.Errorf("test setup error: failed to parse base URL %s: %w", tt.topicDetails.URL, errBase)
					}
					refURL, errRef := url.Parse("fetcherror2_page2")
					if errRef != nil {
						return "", fmt.Errorf("test setup error: failed to parse ref URL %s: %w", "fetcherror2_page2", errRef)
					}
					expectedSecondPageURLForFetchError2 = parsedBase.ResolveReference(refURL).String()
				}

				if tt.name == "fetch error on second page" && fetchURL == expectedSecondPageURLForFetchError2 {
					return "", fmt.Errorf("mock fetch error for second page")
				}
				if html, ok := tt.mockHTML[fetchURL]; ok {
					return html, nil
				}
				if tt.name == "multi page topic" {
					return "<html><body>Generic mock page</body></html>", nil
				}
				return "", fmt.Errorf("unexpected URL fetched in test: %s for test case %s", fetchURL, tt.name)
			}

			gotNavInfos, err := GetTopicPageURLs(tt.topicDetails, 0)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTopicPageURLs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(gotNavInfos, tt.wantNavInfos) {
				t.Errorf("GetTopicPageURLs() got = %v, want %v", gotNavInfos, tt.wantNavInfos)
			}
		})
	}
}

// TODO: Add unit tests for navigation logic here
