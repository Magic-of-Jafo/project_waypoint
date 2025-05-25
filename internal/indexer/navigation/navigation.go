package navigation

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// FetchHTML fetches the HTML content from the given URL.
// It returns the HTML content as a string and an error if any occurred.
func FetchHTML(url string) (string, error) {
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
			fmt.Printf("Warning: Could not parse pagination link href '%s': %v\n", href, err)
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

// TODO: Implement sub-forum page navigation logic here
// This will include functions to:
// - Fetch HTML content (AC1)
// - Parse HTML for pagination links (AC2, AC3, AC4, AC5)
// - Generate ordered list of page URLs (AC6)
// - Handle various pagination scenarios (AC7, AC8)
// - Log errors/messages (AC9)
