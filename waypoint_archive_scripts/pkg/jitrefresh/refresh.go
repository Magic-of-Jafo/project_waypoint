package jitrefresh

import (
	"log"
	"time"

	// "project-waypoint/internal/indexer/navigation" // Using the Epic 1 navigation package
	// "project-waypoint/internal/indexer/topic"      // Using the Epic 1 topic package

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/data"
	// New import
)

// FetchHTMLer defines the signature for a function that fetches HTML content.
// UserAgent is included as it was added to htmlutil.FetchHTML.
// delay is time.Duration
// userAgent is string
// Returns html string and error
type FetchHTMLer func(pageURL string, delay time.Duration, userAgent string) (string, error)

// ParsePaginationLinker defines the signature for a function that parses pagination links.
// htmlContent is string
// pageURL is string
// Returns slice of strings (links) and error
type ParsePaginationLinker func(htmlContent string, pageURL string) ([]string, error)

// ExtractTopicser defines the signature for a function that extracts topics from HTML.
// htmlContent is string
// pageURL is string
// subForumID is string
// Returns slice of data.Topic and error
type ExtractTopicser func(htmlContent string, pageURL string, subForumID string) ([]data.Topic, error)

// PerformJITRefresh fetches the first few pages of a sub-forum, parses live topic IDs,
// compares them against the loaded index, and returns newly discovered topics.
func PerformJITRefresh(
	subForumData data.SubForum,
	cfg *config.Config,
	fetcher FetchHTMLer,
	parser ParsePaginationLinker,
	extractor ExtractTopicser,
) ([]data.Topic, error) {

	log.Printf("[INFO] JIT REFRESH: Starting for SubForum: %s (ID: %s, URL: %s), JITRefreshPages: %d",
		subForumData.Name, subForumData.ID, subForumData.URL, cfg.JITRefreshPages)

	if subForumData.URL == "" {
		log.Printf("[WARNING] JIT REFRESH: SubForum %s (ID: %s) has no URL. Skipping JIT refresh.", subForumData.Name, subForumData.ID)
		return []data.Topic{}, nil
	}

	if cfg.JITRefreshPages <= 0 {
		log.Printf("[INFO] JIT REFRESH: JITRefreshPages is %d for SubForum %s. Skipping JIT scan.", cfg.JITRefreshPages, subForumData.ID)
		return []data.Topic{}, nil
	}

	var allLiveTopics []data.Topic
	scannedPageCount := 0

	// Process the initial page first
	log.Printf("[DEBUG] JIT REFRESH: Fetching and processing initial page: %s", subForumData.URL)
	initialPageHTML, err := fetcher(subForumData.URL, cfg.PolitenessDelay, cfg.UserAgent)
	if err != nil {
		log.Printf("[ERROR] JIT REFRESH: Failed to fetch initial page %s for sub-forum %s: %v", subForumData.URL, subForumData.ID, err)
		return nil, err // Critical if the first page can't be fetched
	}

	liveTopicsOnInitialPage, err := extractor(initialPageHTML, subForumData.URL, subForumData.ID)
	if err != nil {
		log.Printf("[WARNING] JIT REFRESH: Failed to extract topics from initial page %s for sub-forum %s: %v. Continuing with pagination scan if possible.", subForumData.URL, subForumData.ID, err)
		// Not returning error here, as pagination might still yield results from other pages
	} else {
		log.Printf("[DEBUG] JIT REFRESH: Found %d topics on initial page %s", len(liveTopicsOnInitialPage), subForumData.URL)
		allLiveTopics = append(allLiveTopics, liveTopicsOnInitialPage...)
	}
	scannedPageCount++ // Count the initial page

	// Only proceed with pagination if JITRefreshPages allows for more pages
	if cfg.JITRefreshPages > 0 && scannedPageCount >= cfg.JITRefreshPages {
		log.Printf("[INFO] JIT REFRESH: Reached JITRefreshPages limit (%d) after processing initial page for sub-forum %s. Stopping scan.", cfg.JITRefreshPages, subForumData.ID)
	} else {
		log.Printf("[DEBUG] JIT REFRESH: Parsing pagination links from initial page HTML of %s", subForumData.URL)
		subForumPageURLs, err := parser(initialPageHTML, subForumData.URL)
		if err != nil {
			log.Printf("[ERROR] JIT REFRESH: Failed to parse pagination links for sub-forum %s (URL: %s): %v. Proceeding with topics found so far (if any).", subForumData.ID, subForumData.URL, err)
			// Don't return error, as we might have topics from the initial page.
		} else {
			log.Printf("[DEBUG] JIT REFRESH: Found %d pagination links. Processing them.", len(subForumPageURLs))
			for i, pageURL := range subForumPageURLs {
				if cfg.JITRefreshPages > 0 && scannedPageCount >= cfg.JITRefreshPages {
					log.Printf("[INFO] JIT REFRESH: Reached JITRefreshPages limit (%d) for sub-forum %s. Stopping scan of further paginated pages.", cfg.JITRefreshPages, subForumData.ID)
					break
				}
				// Avoid re-processing the initial page if the parser somehow includes it
				if pageURL == subForumData.URL {
					log.Printf("[DEBUG] JIT REFRESH: Skipping page %s as it's the initial page (already processed).", pageURL)
					continue
				}

				log.Printf("[DEBUG] JIT REFRESH: Scanning paginated page %d/%d (URL: %s) for sub-forum %s", i+1, len(subForumPageURLs), pageURL, subForumData.ID)
				scannedPageCount++

				pageHTML, err := fetcher(pageURL, cfg.PolitenessDelay, cfg.UserAgent)
				if err != nil {
					log.Printf("[WARNING] JIT REFRESH: Failed to fetch page %s for sub-forum %s: %v. Skipping page.", pageURL, subForumData.ID, err)
					continue
				}

				liveTopicsOnPage, err := extractor(pageHTML, pageURL, subForumData.ID)
				if err != nil {
					log.Printf("[WARNING] JIT REFRESH: Failed to extract topics from page %s for sub-forum %s: %v. Skipping page.", pageURL, subForumData.ID, err)
					continue
				}

				log.Printf("[DEBUG] JIT REFRESH: Found %d topics on page %s", len(liveTopicsOnPage), pageURL)
				allLiveTopics = append(allLiveTopics, liveTopicsOnPage...)
			}
		}
	}

	existingTopicIDs := make(map[string]struct{})
	for _, existingTopic := range subForumData.Topics {
		existingTopicIDs[existingTopic.ID] = struct{}{}
	}

	newTopics := []data.Topic{}
	seenNewTopicIDs := make(map[string]struct{}) // To de-duplicate topics found across JIT scanned pages

	for _, liveTopic := range allLiveTopics {
		if _, existsInOriginal := existingTopicIDs[liveTopic.ID]; !existsInOriginal {
			if _, existsInNew := seenNewTopicIDs[liveTopic.ID]; !existsInNew {
				newTopics = append(newTopics, liveTopic)
				seenNewTopicIDs[liveTopic.ID] = struct{}{}
				log.Printf("[INFO] JIT REFRESH: Discovered new topic for %s: ID %s, Title: %s", subForumData.ID, liveTopic.ID, liveTopic.Title)
			}
		}
	}

	log.Printf("[INFO] JIT REFRESH: Completed for SubForum %s. Discovered %d new topics from %d scanned pages.", subForumData.ID, len(newTopics), scannedPageCount)
	return newTopics, nil
}
