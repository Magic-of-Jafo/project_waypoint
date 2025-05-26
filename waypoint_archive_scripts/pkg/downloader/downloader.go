package downloader

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"waypoint_archive_scripts/pkg/config"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Downloader struct will hold any persistent configuration or state for the downloader,
// such as the HTTP client or common headers.
type Downloader struct {
	Client          *http.Client
	UserAgent       string
	PolitenessDelay time.Duration
}

// NewDownloader creates and returns a new Downloader instance.
// It takes the application configuration to set up the User-Agent and PolitenessDelay.
func NewDownloader(cfg *config.Config) *Downloader {
	return &Downloader{
		Client: &http.Client{
			Timeout: 30 * time.Second, // Sensible default timeout
		},
		UserAgent:       cfg.UserAgent,
		PolitenessDelay: cfg.PolitenessDelay,
	}
}

// FetchPage downloads the raw HTML content for a given URL.
// It respects the politeness delay and uses the configured User-Agent.
// It handles character encoding based on HTTP headers or defaults to UTF-8.
// Returns the raw HTML as a byte slice and an error if any occurs.
func (d *Downloader) FetchPage(url string) ([]byte, error) {
	// AC8: Respect politeness delay
	if d.PolitenessDelay > 0 {
		log.Printf("Applying politeness delay of %v before fetching %s", d.PolitenessDelay, url)
		time.Sleep(d.PolitenessDelay)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request for URL %s: %v", url, err)
		return nil, err // AC6
	}

	// AC8: Set custom User-Agent
	if d.UserAgent != "" {
		req.Header.Set("User-Agent", d.UserAgent)
	} else {
		// Explicitly set an empty User-Agent if none is configured
		// to prevent Go's default HTTP client from adding its own.
		req.Header.Set("User-Agent", "")
	}

	// AC1: Execute HTTP GET request
	resp, err := d.Client.Do(req)
	if err != nil {
		log.Printf("Error fetching URL %s: %v", url, err) // AC6: Network-related issues
		return nil, err
	}
	defer resp.Body.Close()

	// AC7: Handle HTTP error status codes
	if resp.StatusCode >= 400 {
		log.Printf("HTTP error for URL %s: Status %s", url, resp.Status)
		// Note: Retry logic as per Story 2.5 will be handled by the calling orchestrator or a higher-level retry mechanism.
		// This function focuses on the download attempt and reporting the outcome.
		return nil, &HTTPError{StatusCode: resp.StatusCode, URL: url}
	}

	// AC2: Retrieve full HTTP response
	// AC3, AC5: Extract raw HTML content and handle character encoding
	contentType := resp.Header.Get("Content-Type")
	var bodyReader io.Reader = resp.Body

	// Determine encoding from Content-Type header
	e, name, certain := charset.DetermineEncoding(nil, contentType)
	if !certain && name != "utf-8" { // If not certain and not already utf-8 (common default)
		log.Printf("Encoding for %s (Content-Type: %s) is uncertain (detected: %s). Attempting to read raw bytes.", url, contentType, name)
		// Fallback for uncertain encoding: read raw bytes without transformation
		// This fulfills AC5's requirement to capture raw byte stream faithfully if precise decoding is uncertain.
	} else if e != nil && e != unicode.UTF8 { // If an encoding is determined and it's not UTF-8, transform.
		log.Printf("Decoding %s from %s (Content-Type: %s)", url, name, contentType)
		bodyReader = transform.NewReader(resp.Body, e.NewDecoder())
	} else {
		// If UTF-8 or no specific encoding detected, assume UTF-8 or that raw bytes are fine.
		log.Printf("Reading %s as UTF-8 or raw bytes (Content-Type: %s, Detected Encoding: %s, Certain: %t)", url, contentType, name, certain)
	}

	rawHTML, err := io.ReadAll(bodyReader)
	if err != nil {
		log.Printf("Error reading response body for URL %s: %v", url, err)
		return nil, err
	}

	// AC4: Ensure downloaded HTML is preserved exactly as received (handled by reading directly)
	// AC9: Return raw HTML content
	return rawHTML, nil
}

// HTTPError represents an error related to an HTTP status code.
type HTTPError struct {
	StatusCode int
	URL        string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d fetching URL %s", e.StatusCode, e.URL)
}
