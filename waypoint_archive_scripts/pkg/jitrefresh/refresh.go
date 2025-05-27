package jitrefresh

import (
	"log"
	"time"

	// "project-waypoint/internal/indexer/navigation" // Using the Epic 1 navigation package
	// "project-waypoint/internal/indexer/topic"      // Using the Epic 1 topic package

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/data"
	"waypoint_archive_scripts/pkg/htmlutil" // New import
	"waypoint_archive_scripts/pkg/state"    // For ShouldPerformJITRefresh
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

// ShouldPerformJITRefresh determines if a JIT refresh should be performed for a sub-forum.
// It checks if JIT refresh is enabled (jitRefreshPages > 0) and if the JITRefreshInterval
// has passed since the last attempt for this sub-forum.
func ShouldPerformJITRefresh(
	subForum data.SubForum,
	currentState *state.ArchiveProgressState,
	jitEnabled bool, // True if cfg.JITRefreshPages > 0
	jitInterval time.Duration,
) bool {
	if !jitEnabled {
		log.Printf("[DEBUG] JIT REFRESH: Skipping JIT for SubForum %s, JITRefreshPages is not positive.", subForum.ID)
		return false
	}

	if currentState == nil || currentState.JITRefreshAttempts == nil {
		log.Printf("[INFO] JIT REFRESH: No previous JIT attempt state for SubForum %s. Performing refresh.", subForum.ID)
		return true // No record of last attempt, or state is nil, so refresh.
	}

	lastAttemptTime, ok := currentState.JITRefreshAttempts[subForum.ID]
	if !ok {
		log.Printf("[INFO] JIT REFRESH: No JIT refresh attempt recorded for SubForum %s. Performing refresh.", subForum.ID)
		return true // No attempt recorded for this specific sub-forum.
	}

	if time.Since(lastAttemptTime) >= jitInterval {
		log.Printf("[INFO] JIT REFRESH: JIT refresh interval (%s) has passed for SubForum %s (last attempt: %s). Performing refresh.", jitInterval, subForum.ID, lastAttemptTime)
		return true
	}

	log.Printf("[DEBUG] JIT REFRESH: JIT refresh interval (%s) has NOT passed for SubForum %s (last attempt: %s). Skipping refresh.", jitInterval, subForum.ID, lastAttemptTime)
	return false
}

// PerformJITRefresh fetches the first few pages of a sub-forum, parses live topic IDs,
// compares them against the loaded index, and returns newly discovered topics.
// It now uses interfaces from the htmlutil package.
func PerformJITRefresh(
	subForumData data.SubForum,
	cfg *config.Config, // cfg is used for PolitenessDelay, UserAgent, JITRefreshPages
	fetcher htmlutil.FetchHTMLer,
	parser htmlutil.ParsePaginationLinker,
	extractor htmlutil.ExtractTopicser,
) ([]data.Topic, error) {

	log.Printf("[INFO] JIT REFRESH: Starting for SubForum: %s (ID: %s, URL: %s), JITRefreshPages: %d",
		subForumData.Name, subForumData.ID, subForumData.URL, cfg.JITRefreshPages)

	if subForumData.URL == "" {
		log.Printf("[WARNING] JIT REFRESH: SubForum %s (ID: %s) has no URL. Skipping JIT refresh.", subForumData.Name, subForumData.ID)
		return []data.Topic{}, nil
	}

	if cfg.JITRefreshPages <= 0 {
		// This check is technically redundant if ShouldPerformJITRefresh is called first,
		// but good for robustness if PerformJITRefresh is called directly.
		log.Printf("[INFO] JIT REFRESH: JITRefreshPages is %d for SubForum %s. Skipping JIT scan.", cfg.JITRefreshPages, subForumData.ID)
		return []data.Topic{}, nil
	}

	var allLiveTopics []data.Topic
	scannedPageCount := 0

	// Process the initial page first
	log.Printf("[DEBUG] JIT REFRESH: Fetching and processing initial page: %s", subForumData.URL)
	// Use the FetchHTML method from the fetcher interface
	initialPageHTML, err := fetcher.FetchHTML(subForumData.URL)
	if err != nil {
		log.Printf("[ERROR] JIT REFRESH: Failed to fetch initial page %s for sub-forum %s: %v", subForumData.URL, subForumData.ID, err)
		return nil, err // Critical if the first page can't be fetched
	}

	// Use the ExtractTopics method from the extractor interface
	liveTopicsOnInitialPage, err := extractor.ExtractTopics(initialPageHTML, subForumData.URL, subForumData.ID)
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
		// Use the ParsePaginationLinks method from the parser interface
		subForumPageURLs, err := parser.ParsePaginationLinks(initialPageHTML, subForumData.URL)
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

				// Use FetchHTML method
				pageHTML, err := fetcher.FetchHTML(pageURL)
				if err != nil {
					log.Printf("[WARNING] JIT REFRESH: Failed to fetch page %s for sub-forum %s: %v. Skipping page.", pageURL, subForumData.ID, err)
					continue
				}

				// Use ExtractTopics method
				liveTopicsOnPage, err := extractor.ExtractTopics(pageHTML, pageURL, subForumData.ID)
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
