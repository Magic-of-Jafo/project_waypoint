package jitrefresh

import (
	"bytes"
	"fmt"
	"log"
	"net/url"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/data"

	"github.com/PuerkitoBio/goquery"
)

var mockUtilResponses map[string]string
var mockUtilFetchError error
var mockUtilParsePaginationError error
var mockUtilExtractTopicsError error
var mockUtilExtractTopicsErrorOnURL string // Used to trigger error only for a specific URL

// Adapter for htmlutil.FetchHTMLer
type mockFetcherAdapter struct {
	fetchFunc func(pageURL string, delay time.Duration, userAgent string) (string, error)
	delay     time.Duration
	userAgent string
}

func (m *mockFetcherAdapter) FetchHTML(pageURL string) (string, error) {
	return m.fetchFunc(pageURL, m.delay, m.userAgent)
}

// Adapter for htmlutil.ParsePaginationLinker
type mockParserAdapter struct {
	parseFunc func(htmlContent string, pageURL string) ([]string, error)
}

func (m *mockParserAdapter) ParsePaginationLinks(htmlContent string, pageURL string) ([]string, error) {
	return m.parseFunc(htmlContent, pageURL)
}

// Adapter for htmlutil.ExtractTopicser
type mockExtractorAdapter struct {
	extractFunc func(htmlContent string, pageURL string, subForumID string) ([]data.Topic, error)
}

func (m *mockExtractorAdapter) ExtractTopics(htmlContent string, pageURL string, subForumID string) ([]data.Topic, error) {
	// The mock function mockUtilExtractTopicsFromHTMLInUtil expects subForumID as a string.
	// The interface htmlutil.ExtractTopicser also provides it as a string now (corrected).
	return m.extractFunc(htmlContent, pageURL, subForumID)
}

// Mock implementations remain the same
func mockUtilFetchHTML(pageURL string, delay time.Duration, userAgent string) (string, error) {
	if mockUtilFetchError != nil {
		return "", mockUtilFetchError
	}
	if content, ok := mockUtilResponses[pageURL]; ok {
		log.Printf("[TEST-MOCK] mockUtilFetchHTML called for URL: %s, UserAgent: %s, returning stored content.", pageURL, userAgent)
		return content, nil
	}
	log.Printf("[TEST-MOCK] mockUtilFetchHTML called for URL: %s, UserAgent: %s, but no mock response found. Returning error.", pageURL, userAgent)
	return "", fmt.Errorf("mockUtilFetchHTML: no response for %s", pageURL)
}

func mockUtilParsePaginationLinks(htmlContent string, pageURL string) ([]string, error) {
	log.Printf("[TEST-MOCK] mockUtilParsePaginationLinks called for URL: %s. mockUtilParsePaginationError is: %v", pageURL, mockUtilParsePaginationError)
	if mockUtilParsePaginationError != nil {
		log.Printf("[TEST-MOCK] mockUtilParsePaginationLinks returning error: %v", mockUtilParsePaginationError)
		return nil, mockUtilParsePaginationError
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("mockUtilParsePaginationLinks: goquery failed for %s: %w", pageURL, err)
	}

	parsedPageURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("mockUtilParsePaginationLinks: failed to parse page URL %s: %w", pageURL, err)
	}

	var links []string
	seenLinks := make(map[string]bool)

	selector := "div.pagination a[href], .pagmenu a[href], .page-nav a[href], .nav-links a[href]"
	log.Printf("[TEST-MOCK-DEBUG] mockUtilParsePaginationLinks: Using selector: %s for URL: %s", selector, pageURL)
	doc.Find(selector).Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		text := s.Text()
		log.Printf("[TEST-MOCK-DEBUG] mockUtilParsePaginationLinks: Found element with href: '%s' (exists: %t), text: '%s'", href, exists, text)
		if !exists || href == "" || href == "#" || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			return
		}
		absURL, err := parsedPageURL.Parse(href)
		if err != nil {
			log.Printf("[TEST-MOCK-WARNING] mockUtilParsePaginationLinks: Error parsing pagination link '%s' on page %s: %v", href, pageURL, err)
			return
		}
		absLinkStr := absURL.String()
		if !seenLinks[absLinkStr] {
			links = append(links, absLinkStr)
			seenLinks[absLinkStr] = true
		}
	})

	sort.Strings(links)

	log.Printf("[TEST-MOCK] mockUtilParsePaginationLinks for %s, parsed %d unique links from HTML content.", pageURL, len(links))
	return links, nil
}

func mockUtilExtractTopicsFromHTMLInUtil(htmlContent string, pageURL string, subForumID string) ([]data.Topic, error) {
	log.Printf("[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for %s, subForumID %s, mockUtilExtractTopicsError: %v, mockUtilExtractTopicsErrorOnURL: %s", pageURL, subForumID, mockUtilExtractTopicsError, mockUtilExtractTopicsErrorOnURL)
	if mockUtilExtractTopicsError != nil {
		if mockUtilExtractTopicsErrorOnURL == "" || mockUtilExtractTopicsErrorOnURL == pageURL {
			log.Printf("[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil returning error: %v for URL %s", mockUtilExtractTopicsError, pageURL)
			return nil, mockUtilExtractTopicsError
		}
	}

	var topics []data.Topic
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("[TEST-MOCK-ERROR] mockUtilExtractTopicsFromHTMLInUtil failed to parse HTML from %s: %v", pageURL, err)
		return nil, fmt.Errorf("mockUtilExtractTopicsFromHTMLInUtil failed to parse HTML: %w", err)
	}

	// Simplified mock extraction: look for <a href="topic.php?id=TID">Title</a>
	doc.Find("a[href*='topic.php?id=']").Each(func(i int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		topicURL, _ := url.Parse(href)
		topicID := topicURL.Query().Get("id")
		title := strings.TrimSpace(s.Text()) // Use the anchor text as title
		if topicID != "" {
			// The mock extractor uses string subForumID. The adapter handles int to string conversion.
			topics = append(topics, data.Topic{ID: topicID, SubForumID: subForumID, Title: title, URL: href})
		}
	})

	log.Printf("[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for %s, subForumID %s, extracted %d topics based on content patterns.", pageURL, subForumID, len(topics))
	return topics, nil
}

func TestPerformJITRefresh(t *testing.T) {
	log.SetOutput(os.Stderr) // Ensure logs go to a place we can see during tests, if needed

	tests := []struct {
		name                string
		subForumData        data.SubForum
		cfg                 *config.Config
		mockHTTPResponses   map[string]string // URL -> HTML content
		mockFetchErr        error
		mockParseErr        error
		mockExtractErr      error
		mockExtractErrOnURL string
		wantNewTopics       []data.Topic
		wantErr             bool
		expectedLogs        []string
	}{
		// --- Scenario 1: Successful discovery of new topics ---
		{
			name: "discover new topics",
			subForumData: data.SubForum{
				ID:   "1",
				Name: "SubForum One",
				URL:  "http://forum.example.com/sf1",
				Topics: []data.Topic{
					{ID: "t100", SubForumID: "1", Title: "Existing Topic 100"},
				},
			},
			cfg: &config.Config{
				JITRefreshPages: 2,
				PolitenessDelay: 0,
				ForumBaseURL:    "http://forum.example.com",
				UserAgent:       "TestAgent/1.0",
			},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf1": `<html><body>` +
					`<!-- Topic link for t101 --> <a href="topic.php?id=t101">New Topic 101</a>` +
					`<!-- Topic link for t100 --> <a href="topic.php?id=t100">Existing Topic 100</a>` +
					`<div class="pagination"><a href="http://forum.example.com/sf1?page=2">Next</a></div>` +
					`</body></html>`,
				"http://forum.example.com/sf1?page=2": `<html><body>` +
					`<!-- Topic link for t102 --> <a href="topic.php?id=t102">New Topic 102 From Page 2</a>` +
					`<!-- No more pagination -->` +
					`</body></html>`,
			},
			wantNewTopics: []data.Topic{
				{ID: "t101", SubForumID: "1", Title: "New Topic 101", URL: "topic.php?id=t101"},
				{ID: "t102", SubForumID: "1", Title: "New Topic 102 From Page 2", URL: "topic.php?id=t102"},
			},
			wantErr: false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SubForum One (ID: 1, URL: http://forum.example.com/sf1), JITRefreshPages: 2",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sf1",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf1",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf1, subForumID 1, extracted 2 topics",
				"[DEBUG] JIT REFRESH: Found 2 topics on initial page http://forum.example.com/sf1",
				"[DEBUG] JIT REFRESH: Parsing pagination links from initial page HTML of http://forum.example.com/sf1",
				"[TEST-MOCK] mockUtilParsePaginationLinks for http://forum.example.com/sf1, parsed 1 unique links from HTML content.",
				"[DEBUG] JIT REFRESH: Found 1 pagination links. Processing them.",
				"[DEBUG] JIT REFRESH: Scanning paginated page 1/1 (URL: http://forum.example.com/sf1?page=2) for sub-forum 1",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf1?page=2",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf1?page=2, subForumID 1, extracted 1 topics",
				"[DEBUG] JIT REFRESH: Found 1 topics on page http://forum.example.com/sf1?page=2",
				"[INFO] JIT REFRESH: Discovered new topic for 1: ID t101, Title: New Topic 101",
				"[INFO] JIT REFRESH: Discovered new topic for 1: ID t102, Title: New Topic 102 From Page 2",
				"[INFO] JIT REFRESH: Completed for SubForum 1. Discovered 2 new topics from 2 scanned pages.",
			},
		},
		// --- Scenario 2: No new topics found ---
		{
			name: "no new topics found",
			subForumData: data.SubForum{
				ID:     "2",
				Name:   "SubForum Two",
				URL:    "http://forum.example.com/sf2",
				Topics: []data.Topic{{ID: "t200", SubForumID: "2", Title: "Existing Topic 200 Link"}},
			},
			cfg: &config.Config{JITRefreshPages: 1, PolitenessDelay: 0, ForumBaseURL: "http://forum.example.com", UserAgent: "TestAgent/1.0"},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf2": `<html><body><a href="topic.php?id=t200">Existing Topic 200 Link</a></body></html>`,
			},
			wantNewTopics: []data.Topic{},
			wantErr:       false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SubForum Two (ID: 2, URL: http://forum.example.com/sf2), JITRefreshPages: 1",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sf2",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf2",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf2, subForumID 2, extracted 1 topics",
				"[DEBUG] JIT REFRESH: Found 1 topics on initial page http://forum.example.com/sf2",
				"[INFO] JIT REFRESH: Reached JITRefreshPages limit (1) after processing initial page for sub-forum 2. Stopping scan.",
				"[INFO] JIT REFRESH: Completed for SubForum 2. Discovered 0 new topics from 1 scanned pages.",
			},
		},
		// --- Scenario 3: Sub-forum URL is empty ---
		{
			name:          "sub-forum URL empty",
			subForumData:  data.SubForum{ID: "3", Name: "SubForum Three", URL: ""},
			cfg:           &config.Config{JITRefreshPages: 1, UserAgent: "TestAgent/1.0"},
			wantNewTopics: []data.Topic{},
			wantErr:       false,
			expectedLogs:  []string{"WARNING] JIT REFRESH: SubForum SubForum Three (ID: 3) has no URL. Skipping JIT refresh."},
		},
		// --- Scenario 4: JITRefreshPages is 0 ---
		{
			name:         "JITRefreshPages is 0",
			subForumData: data.SubForum{ID: "4", Name: "SubForum Four", URL: "http://forum.example.com/sf4"},
			cfg:          &config.Config{JITRefreshPages: 0, PolitenessDelay: 0, ForumBaseURL: "http://forum.example.com", UserAgent: "TestAgent/1.0"},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf4": `<html><body><a href="topic.php?id=t400">Topic 400</a></body></html>`,
			},
			wantNewTopics: []data.Topic{},
			wantErr:       false,
			expectedLogs:  []string{"INFO] JIT REFRESH: JITRefreshPages is 0 for SubForum 4. Skipping JIT scan."},
		},
		// --- Scenario 5: Error fetching initial page for pagination ---
		{
			name:          "error fetching initial page",
			subForumData:  data.SubForum{ID: "5", Name: "SubForum Five", URL: "http://forum.example.com/sf5_fetch_error"},
			cfg:           &config.Config{JITRefreshPages: 1, PolitenessDelay: 0, UserAgent: "TestAgent/1.0"},
			mockFetchErr:  fmt.Errorf("simulated fetch error for initial page"),
			wantNewTopics: nil,
			wantErr:       true,
			expectedLogs:  []string{"ERROR] JIT REFRESH: Failed to fetch initial page http://forum.example.com/sf5_fetch_error"},
		},
		// --- Scenario 6: Error parsing pagination links ---
		{
			name:         "error parsing pagination links",
			subForumData: data.SubForum{ID: "6", Name: "SubForum Six", URL: "http://forum.example.com/sf6_parse_error"},
			cfg:          &config.Config{JITRefreshPages: 1, PolitenessDelay: 0, UserAgent: "TestAgent/1.0"},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf6_parse_error": `<html><body><a href="topic.php?id=t600">Topic 600</a></body></html>`,
			},
			mockParseErr:  fmt.Errorf("simulated parse pagination error"),
			wantNewTopics: []data.Topic{{ID: "t600", SubForumID: "6", Title: "Topic 600", URL: "topic.php?id=t600"}},
			wantErr:       false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SubForum Six (ID: 6, URL: http://forum.example.com/sf6_parse_error), JITRefreshPages: 1",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sf6_parse_error",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf6_parse_error",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf6_parse_error, subForumID 6, extracted 1 topics",
				"[DEBUG] JIT REFRESH: Found 1 topics on initial page http://forum.example.com/sf6_parse_error",
				"[INFO] JIT REFRESH: Reached JITRefreshPages limit (1) after processing initial page for sub-forum 6. Stopping scan.",
				"[INFO] JIT REFRESH: Discovered new topic for 6: ID t600, Title: Topic 600",
				"[INFO] JIT REFRESH: Completed for SubForum 6. Discovered 1 new topics from 1 scanned pages.",
			},
		},
		// --- Scenario 7: Error fetching a subsequent page ---
		{
			name:         "error fetching subsequent page",
			subForumData: data.SubForum{ID: "7", Name: "SF Seven", URL: "http://forum.example.com/sf7"},
			cfg:          &config.Config{JITRefreshPages: 2, PolitenessDelay: 0, UserAgent: "TestAgent/1.0"},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf7": `<html><body><a href="topic.php?id=t700">Topic 700</a><div class="pagination"><a href="http://forum.example.com/sf7?page=2_fetch_error">Next</a></div></body></html>`,
			},
			wantNewTopics: []data.Topic{{ID: "t700", SubForumID: "7", Title: "Topic 700", URL: "topic.php?id=t700"}},
			wantErr:       false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SF Seven (ID: 7, URL: http://forum.example.com/sf7), JITRefreshPages: 2",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sf7",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf7",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf7, subForumID 7, extracted 1 topics",
				"[DEBUG] JIT REFRESH: Found 1 topics on initial page http://forum.example.com/sf7",
				"[DEBUG] JIT REFRESH: Parsing pagination links from initial page HTML of http://forum.example.com/sf7",
				"[TEST-MOCK] mockUtilParsePaginationLinks for http://forum.example.com/sf7, parsed 1 unique links from HTML content.",
				"[DEBUG] JIT REFRESH: Found 1 pagination links. Processing them.",
				"[DEBUG] JIT REFRESH: Scanning paginated page 1/1 (URL: http://forum.example.com/sf7?page=2_fetch_error) for sub-forum 7",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf7?page=2_fetch_error, UserAgent: TestAgent/1.0, but no mock response found. Returning error.",
				"[WARNING] JIT REFRESH: Failed to fetch page http://forum.example.com/sf7?page=2_fetch_error for sub-forum 7: mockUtilFetchHTML: no response for http://forum.example.com/sf7?page=2_fetch_error. Skipping page.",
				"[INFO] JIT REFRESH: Discovered new topic for 7: ID t700, Title: Topic 700",
				"[INFO] JIT REFRESH: Completed for SubForum 7. Discovered 1 new topics from 2 scanned pages.",
			},
		},
		// --- Scenario 8: Error extracting topics from a subsequent page ---
		{
			name:         "error extracting topics from subsequent page",
			subForumData: data.SubForum{ID: "8", Name: "SF Eight Sub", URL: "http://forum.example.com/sf8_sub"},
			cfg:          &config.Config{JITRefreshPages: 2, PolitenessDelay: 0, UserAgent: "TestAgent/1.0"},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf8_sub":                      `<html><body><a href="topic.php?id=t800">Topic 800</a><div class="pagination"><a href="http://forum.example.com/sf8_sub?page=2_extract_error">Next</a></div></body></html>`,
				"http://forum.example.com/sf8_sub?page=2_extract_error": `<html><body>This page will cause an extract error by not having parseable topics. Mock will return error.</body></html>`,
			},
			mockExtractErr:      fmt.Errorf("simulated extract topics error"),
			mockExtractErrOnURL: "http://forum.example.com/sf8_sub?page=2_extract_error",
			wantNewTopics:       []data.Topic{{ID: "t800", SubForumID: "8", Title: "Topic 800", URL: "topic.php?id=t800"}},
			wantErr:             false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SF Eight Sub (ID: 8, URL: http://forum.example.com/sf8_sub), JITRefreshPages: 2",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sf8_sub",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf8_sub",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf8_sub, subForumID 8, extracted 1 topics",
				"[DEBUG] JIT REFRESH: Found 1 topics on initial page http://forum.example.com/sf8_sub",
				"[DEBUG] JIT REFRESH: Parsing pagination links from initial page HTML of http://forum.example.com/sf8_sub",
				"[TEST-MOCK] mockUtilParsePaginationLinks for http://forum.example.com/sf8_sub, parsed 1 unique links from HTML content.",
				"[DEBUG] JIT REFRESH: Found 1 pagination links. Processing them.",
				"[DEBUG] JIT REFRESH: Scanning paginated page 1/1 (URL: http://forum.example.com/sf8_sub?page=2_extract_error) for sub-forum 8",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf8_sub?page=2_extract_error",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf8_sub?page=2_extract_error, subForumID 8, mockUtilExtractTopicsError: simulated extract topics error, mockUtilExtractTopicsErrorOnURL: http://forum.example.com/sf8_sub?page=2_extract_error",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil returning error: simulated extract topics error for URL http://forum.example.com/sf8_sub?page=2_extract_error",
				"[WARNING] JIT REFRESH: Failed to extract topics from page http://forum.example.com/sf8_sub?page=2_extract_error for sub-forum 8: simulated extract topics error. Skipping page.",
				"[INFO] JIT REFRESH: Discovered new topic for 8: ID t800, Title: Topic 800",
				"[INFO] JIT REFRESH: Completed for SubForum 8. Discovered 1 new topics from 2 scanned pages.",
			},
		},
		// --- Scenario 9: Single-page sub-forum ---
		{
			name:         "single-page sub-forum",
			subForumData: data.SubForum{ID: "9", Name: "SubForum Nine", URL: "http://forum.example.com/sf9_single"},
			cfg:          &config.Config{JITRefreshPages: 1, PolitenessDelay: 0, UserAgent: "TestAgent/1.0"},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sf9_single": `<html><body><a href="topic.php?id=t901">SP Topic 901</a><a href="topic.php?id=t902">SP Topic 902</a></body></html>`,
			},
			wantNewTopics: []data.Topic{
				{ID: "t901", SubForumID: "9", Title: "SP Topic 901", URL: "topic.php?id=t901"},
				{ID: "t902", SubForumID: "9", Title: "SP Topic 902", URL: "topic.php?id=t902"},
			},
			wantErr: false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SubForum Nine (ID: 9, URL: http://forum.example.com/sf9_single), JITRefreshPages: 1",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sf9_single",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sf9_single",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sf9_single, subForumID 9, extracted 2 topics",
				"[DEBUG] JIT REFRESH: Found 2 topics on initial page http://forum.example.com/sf9_single",
				"[INFO] JIT REFRESH: Reached JITRefreshPages limit (1) after processing initial page for sub-forum 9. Stopping scan.",
				"[INFO] JIT REFRESH: Discovered new topic for 9: ID t901, Title: SP Topic 901",
				"[INFO] JIT REFRESH: Discovered new topic for 9: ID t902, Title: SP Topic 902",
				"[INFO] JIT REFRESH: Completed for SubForum 9. Discovered 2 new topics from 1 scanned pages.",
			},
		},
		// --- Scenario 10: UserAgent passed to FetchHTML ---
		{
			name:         "UserAgent passed to FetchHTML",
			subForumData: data.SubForum{ID: "10", Name: "SubForum UserAgent", URL: "http://forum.example.com/sfUA"},
			cfg:          &config.Config{JITRefreshPages: 1, UserAgent: "ConfiguredUA/1.1", PolitenessDelay: time.Millisecond * 10},
			mockHTTPResponses: map[string]string{
				"http://forum.example.com/sfUA": `<html><body><a href="topic.php?id=ua1">UA Topic 1</a></body></html>`,
			},
			wantNewTopics: []data.Topic{
				{ID: "ua1", SubForumID: "10", Title: "UA Topic 1", URL: "topic.php?id=ua1"},
			},
			wantErr: false,
			expectedLogs: []string{
				"[INFO] JIT REFRESH: Starting for SubForum: SubForum UserAgent (ID: 10, URL: http://forum.example.com/sfUA), JITRefreshPages: 1",
				"[DEBUG] JIT REFRESH: Fetching and processing initial page: http://forum.example.com/sfUA",
				"[TEST-MOCK] mockUtilFetchHTML called for URL: http://forum.example.com/sfUA, UserAgent: ConfiguredUA/1.1",
				"[TEST-MOCK] mockUtilExtractTopicsFromHTMLInUtil for http://forum.example.com/sfUA, subForumID 10, extracted 1 topics",
				"[DEBUG] JIT REFRESH: Found 1 topics on initial page http://forum.example.com/sfUA",
				"[INFO] JIT REFRESH: Reached JITRefreshPages limit (1) after processing initial page for sub-forum 10. Stopping scan.",
				"[INFO] JIT REFRESH: Discovered new topic for 10: ID ua1, Title: UA Topic 1",
				"[INFO] JIT REFRESH: Completed for SubForum 10. Discovered 1 new topics from 1 scanned pages.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logBuf bytes.Buffer
			log.SetOutput(&logBuf)
			defer func() { log.SetOutput(os.Stderr) }()

			mockUtilResponses = tt.mockHTTPResponses
			mockUtilFetchError = tt.mockFetchErr
			mockUtilParsePaginationError = tt.mockParseErr
			mockUtilExtractTopicsError = tt.mockExtractErr
			mockUtilExtractTopicsErrorOnURL = tt.mockExtractErrOnURL

			// Create adapter instances
			fetcherAdapter := &mockFetcherAdapter{
				fetchFunc: mockUtilFetchHTML,
				delay:     tt.cfg.PolitenessDelay,
				userAgent: tt.cfg.UserAgent,
			}
			parserAdapter := &mockParserAdapter{parseFunc: mockUtilParsePaginationLinks}
			extractorAdapter := &mockExtractorAdapter{extractFunc: mockUtilExtractTopicsFromHTMLInUtil}

			// Call PerformJITRefresh with the adapter instances
			gotNewTopics, err := PerformJITRefresh(tt.subForumData, tt.cfg,
				fetcherAdapter,
				parserAdapter,
				extractorAdapter,
			)

			if (err != nil) != tt.wantErr {
				t.Errorf("PerformJITRefresh() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if gotNewTopics == nil && tt.wantNewTopics != nil {
				t.Errorf("PerformJITRefresh() returned nil gotNewTopics, but wantNewTopics is non-nil %#v", tt.wantNewTopics)
			}

			sortTopics(gotNewTopics)
			sortTopics(tt.wantNewTopics)

			if !reflect.DeepEqual(gotNewTopics, tt.wantNewTopics) {
				t.Errorf("PerformJITRefresh() gotNewTopics = %#v, want %#v", gotNewTopics, tt.wantNewTopics)
			}

			logOutput := logBuf.String()
			for _, expectedLog := range tt.expectedLogs {
				if !strings.Contains(logOutput, expectedLog) {
					t.Errorf("PerformJITRefresh() log output missing: %s\nFull logs:\n%s", expectedLog, logOutput)
				}
			}
		})
	}
}

func sortTopics(topics []data.Topic) {
	sort.Slice(topics, func(i, j int) bool {
		return topics[i].ID < topics[j].ID
	})
}
