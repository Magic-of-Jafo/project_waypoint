package navigation

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"waypoint_archive_scripts/internal/indexer/logger"
	"waypoint_archive_scripts/pkg/data"

	"github.com/PuerkitoBio/goquery"
)

// fetchHTMLFunc is the type for the HTML fetching function
type fetchHTMLFunc func(url string, delay time.Duration) (string, error)

// defaultFetchHTML is the default implementation of fetchHTMLFunc
func defaultFetchHTML(url string, delay time.Duration) (string, error) {
	if delay > 0 {
		logger.Debugf("Politeness delay: sleeping for %v before fetching %s", delay, url)
		time.Sleep(delay)
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get URL %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get URL %s: status code %d", url, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body from %s: %w", url, err)
	}

	return string(bodyBytes), nil
}

// FetchHTML is the function used to fetch HTML content. It can be replaced in tests.
var FetchHTML fetchHTMLFunc = defaultFetchHTML

// ParsePaginationLinks extracts all unique pagination links from HTML content.
// It determines the total number of pages and generates a list of absolute URLs
// for each page in the sub-forum.
func ParsePaginationLinks(htmlContent string, pageURL string) ([]string, error) {
	parsedCurrentURL, err := url.Parse(pageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pageURL '%s': %w", pageURL, err)
	}

	forumID := parsedCurrentURL.Query().Get("forum")
	if forumID == "" {
		return nil, fmt.Errorf("forum ID not found in query parameters of URL: %s", pageURL)
	}

	basePathForLinks := fmt.Sprintf("%s://%s%s", parsedCurrentURL.Scheme, parsedCurrentURL.Host, parsedCurrentURL.Path)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML content: %w", err)
	}

	var startValues []int
	doc.Find("td.normal.bgc1.b.midtext a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		linkText := strings.ToLower(s.Text())
		if linkText == "next" || linkText == "prev" || linkText == "[next]" || linkText == "[prev]" {
			return
		}

		resolvedLinkURL, err := parsedCurrentURL.Parse(href)
		if err != nil {
			logger.Warnf("Could not parse pagination link href '%s': %v", href, err)
			return
		}

		if strings.HasSuffix(resolvedLinkURL.Path, "viewforum.php") || (resolvedLinkURL.Path == "" && strings.HasSuffix(parsedCurrentURL.Path, "viewforum.php")) {
			startStr := resolvedLinkURL.Query().Get("start")
			if startStr != "" {
				startVal, err := strconv.Atoi(startStr)
				if err == nil && startVal >= 0 {
					startValues = append(startValues, startVal)
				}
			}
		}
	})

	topicsPerPage := 30

	maxStart := 0
	if len(startValues) > 0 {
		sort.Ints(startValues)
		maxStart = startValues[len(startValues)-1]
	}

	totalPages := 1
	if topicsPerPage > 0 && maxStart > 0 {
		totalPages = (maxStart / topicsPerPage) + 1
	} else if maxStart == 0 && len(startValues) > 0 {
		// If startValues contains only 0, or is empty, totalPages remains 1.
	}

	var allPageURLs []string
	pageURLsSet := make(map[string]struct{})

	qPage1 := url.Values{}
	qPage1.Set("forum", forumID)
	page1URL := fmt.Sprintf("%s?%s", basePathForLinks, qPage1.Encode())

	if _, ok := pageURLsSet[page1URL]; !ok {
		allPageURLs = append(allPageURLs, page1URL)
		pageURLsSet[page1URL] = struct{}{}
	}

	for i := 1; i < totalPages; i++ {
		currentStartValue := i * topicsPerPage
		q := url.Values{}
		q.Set("forum", forumID)
		q.Set("start", strconv.Itoa(currentStartValue))

		nextPageURL := fmt.Sprintf("%s?%s", basePathForLinks, q.Encode())
		if _, ok := pageURLsSet[nextPageURL]; !ok {
			allPageURLs = append(allPageURLs, nextPageURL)
			pageURLsSet[nextPageURL] = struct{}{}
		}
	}

	if totalPages == 1 && len(allPageURLs) == 0 {
		allPageURLs = append(allPageURLs, page1URL)
	}

	return allPageURLs, nil
}

// PageNavigationInfo holds information about a single page within a topic.
type PageNavigationInfo struct {
	PageNumber int    // 1-indexed page number
	URL        string // Absolute URL of the page
}

// GetTopicPageURLs fetches and parses a topic's pages to return all page URLs.
// It handles pagination by following "Next" links or page numbers until the last page is found.
// It now also accepts a politenessDelay to be passed to FetchHTML.
func GetTopicPageURLs(topicDetails data.Topic, politenessDelay time.Duration) ([]PageNavigationInfo, error) {
	logger.Infof("Starting to get page URLs for Topic ID: %s (URL: %s), Politeness Delay: %v", topicDetails.ID, topicDetails.URL, politenessDelay)

	// Ensure we have a valid topic ID
	if topicDetails.ID == "" {
		logger.Errorf("GetTopicPageURLs: Topic ID cannot be empty for TopicDetails: %+v", topicDetails)
		return nil, fmt.Errorf("topic ID cannot be empty")
	}

	// Use the topic's URL as the starting point
	firstPageURL, err := url.Parse(topicDetails.URL)
	if err != nil {
		logger.Errorf("GetTopicPageURLs: Failed to parse topic URL '%s' for Topic ID %s: %v", topicDetails.URL, topicDetails.ID, err)
		return nil, fmt.Errorf("failed to parse topic URL '%s': %w", topicDetails.URL, err)
	}
	if firstPageURL.Scheme == "" || firstPageURL.Host == "" {
		logger.Errorf("GetTopicPageURLs: Invalid topic URL '%s' (missing scheme or host) for Topic ID %s", topicDetails.URL, topicDetails.ID)
		return nil, fmt.Errorf("invalid topic URL '%s': missing scheme or host", topicDetails.URL)
	}

	currentURL := firstPageURL.String()
	var pageNavInfos []PageNavigationInfo
	seenURLs := make(map[string]struct{})
	pageCounter := 0

	for {
		if _, seen := seenURLs[currentURL]; !seen {
			pageCounter++
			pageNavInfos = append(pageNavInfos, PageNavigationInfo{PageNumber: pageCounter, URL: currentURL})
			seenURLs[currentURL] = struct{}{}
			logger.Debugf("GetTopicPageURLs: Found page %d for Topic ID %s: %s", pageCounter, topicDetails.ID, currentURL) // DEBUG level for each page
		}

		html, err := FetchHTML(currentURL, politenessDelay)
		if err != nil {
			logger.Errorf("GetTopicPageURLs: Failed to fetch page %s for Topic ID %s: %v", currentURL, topicDetails.ID, err)
			return pageNavInfos, fmt.Errorf("failed to fetch page %s: %w", currentURL, err)
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
		if err != nil {
			logger.Errorf("GetTopicPageURLs: Failed to parse HTML for page %s (Topic ID %s): %v", currentURL, topicDetails.ID, err)
			return pageNavInfos, fmt.Errorf("failed to parse HTML for %s: %w", currentURL, err)
		}

		var nextLink string
		doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
			text := strings.ToLower(strings.TrimSpace(s.Text()))
			if text == "next" || text == "[next]" {
				if href, exists := s.Attr("href"); exists {
					nextLink = href
				}
			}
		})

		if nextLink == "" {
			logger.Debugf("GetTopicPageURLs: No 'Next' link found on page %s for Topic ID %s. Assuming last page.", currentURL, topicDetails.ID)
			break
		}

		parsedCurrent, err := url.Parse(currentURL)
		if err != nil {
			logger.Errorf("GetTopicPageURLs: Failed to parse currentURL '%s' for relative resolution (Topic ID %s): %v", currentURL, topicDetails.ID, err)
			return pageNavInfos, fmt.Errorf("failed to parse currentURL for relative resolution '%s': %w", currentURL, err)
		}
		resolvedNextURL, err := parsedCurrent.Parse(nextLink)
		if err != nil {
			logger.Errorf("GetTopicPageURLs: Failed to parse nextLink '%s' relative to '%s' (Topic ID %s): %v", nextLink, currentURL, topicDetails.ID, err)
			return pageNavInfos, fmt.Errorf("failed to parse next URL '%s' relative to '%s': %w", nextLink, currentURL, err)
		}
		nextURL := resolvedNextURL.String()

		if _, seen := seenURLs[nextURL]; seen {
			logger.Warnf("GetTopicPageURLs: Detected pagination loop or revisit at URL: %s for Topic ID %s. Stopping pagination here.", nextURL, topicDetails.ID)
			break
		}

		currentURL = nextURL
	}

	if len(pageNavInfos) == 0 && topicDetails.URL != "" {
		pageNavInfos = append(pageNavInfos, PageNavigationInfo{PageNumber: 1, URL: firstPageURL.String()})
		logger.Infof("GetTopicPageURLs: No pages were processed but initial URL was valid for Topic ID %s. Returning first page URL: %s", topicDetails.ID, firstPageURL.String())
	}

	logger.Infof("GetTopicPageURLs: Successfully found %d page(s) for Topic ID: %s", len(pageNavInfos), topicDetails.ID)
	return pageNavInfos, nil
}

// TODO: Implement sub-forum page navigation logic here
// This will include functions to:
// - Fetch HTML content (AC1)
// - Parse HTML for pagination links (AC2, AC3, AC4, AC5)
// - Generate ordered list of page URLs (AC6)
// - Handle various pagination scenarios (AC7, AC8)
// - Log errors/messages (AC9)
