package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/data"
	"waypoint_archive_scripts/pkg/downloader"
	"waypoint_archive_scripts/pkg/htmlutil"
	"waypoint_archive_scripts/pkg/indexerlogic"
	"waypoint_archive_scripts/pkg/jitrefresh"
	"waypoint_archive_scripts/pkg/metrics"
	"waypoint_archive_scripts/pkg/state"
	"waypoint_archive_scripts/pkg/storer"
	"waypoint_archive_scripts/pkg/util"
)

// getAllPageURLsForTopic retrieves all unique page URLs for a given topic.
// It fetches the topic's first page, parses pagination links, and collects all URLs.
func getAllPageURLsForTopic(topic data.Topic, cfg *config.Config, fetcher func(pageURL string, delay time.Duration, userAgent string) (string, error), parser func(htmlContent string, pageURL string) ([]string, error)) ([]string, error) {
	log.Printf("[INFO] NAV: Getting all page URLs for Topic ID: %s (%s), Base URL: %s", topic.ID, topic.Title, topic.URL)
	if topic.URL == "" {
		log.Printf("[WARNING] NAV: Topic %s has no base URL. Cannot get page URLs.", topic.ID)
		return nil, fmt.Errorf("topic %s has no base URL", topic.ID)
	}

	// Fetch the first page (base URL of the topic)
	firstPageHTML, err := fetcher(topic.URL, cfg.PolitenessDelay, cfg.UserAgent)
	if err != nil {
		log.Printf("[ERROR] NAV: Failed to fetch first page %s for topic %s: %v", topic.URL, topic.ID, err)
		return nil, fmt.Errorf("failed to fetch first page for topic %s: %w", topic.ID, err)
	}

	// Parse pagination links from the first page
	// Note: ParsePaginationLinks is expected to return absolute URLs
	pageLinks, err := parser(firstPageHTML, topic.URL)
	if err != nil {
		log.Printf("[ERROR] NAV: Failed to parse pagination links from %s for topic %s: %v", topic.URL, topic.ID, err)
		// Proceed with just the base URL if parsing fails but fetching succeeded
		return []string{topic.URL}, nil
	}

	// Collect all unique URLs, ensuring the base URL is included
	uniqueURLs := make(map[string]struct{})
	allPageURLs := []string{}

	// Add the base URL first
	if _, exists := uniqueURLs[topic.URL]; !exists {
		allPageURLs = append(allPageURLs, topic.URL)
		uniqueURLs[topic.URL] = struct{}{}
	}

	// Add links found by the parser
	for _, link := range pageLinks {
		if _, exists := uniqueURLs[link]; !exists {
			allPageURLs = append(allPageURLs, link)
			uniqueURLs[link] = struct{}{}
		}
	}

	// Further sorting or filtering of URLs might be needed here if `parser` is too broad
	// For now, assume all links returned by parser are valid page URLs for the topic.

	log.Printf("[INFO] NAV: Topic %s - found %d unique page URLs.", topic.ID, len(allPageURLs))
	return allPageURLs, nil
}

// downloadTopicPageHTML_Placeholder simulates downloading HTML content for a given page URL.
// It includes placeholders for politeness delay and user agent usage.
func downloadTopicPageHTML_Placeholder(pageURL string, topicID string, pageNum int, cfg *config.Config) (string, error) {
	log.Printf("[INFO] DOWNLOAD_PLACEHOLDER: Attempting to download page %d of Topic ID: %s (URL: %s) using User-Agent: '%s'",
		pageNum, topicID, pageURL, cfg.UserAgent)

	if cfg.PolitenessDelay > 0 {
		log.Printf("[DEBUG] DOWNLOAD_PLACEHOLDER: Applying politeness delay: %v", cfg.PolitenessDelay)
		time.Sleep(cfg.PolitenessDelay)
	}

	// Simulate a download error for a specific URL pattern for testing
	if pageURL == "http://forum.example.com/sf1?page=2_download_error" { // Example error condition
		log.Printf("[ERROR] DOWNLOAD_PLACEHOLDER: Simulated download error for URL: %s", pageURL)
		return "", fmt.Errorf("simulated download error for %s", pageURL)
	}

	// Simulate HTML content
	simulatedHTML := fmt.Sprintf("<html><head><title>Page %d of Topic %s</title></head><body><h1>Content of page %d from %s</h1><p>Mock content.</p></body></html>", pageNum, topicID, pageNum, pageURL)

	log.Printf("[INFO] DOWNLOAD_PLACEHOLDER: Successfully downloaded page %d of Topic ID: %s (URL: %s). Simulated size: %d bytes.",
		pageNum, topicID, pageURL, len(simulatedHTML))
	return simulatedHTML, nil
}

// storePageHTML_Placeholder simulates storing HTML content to a file.
// It includes placeholders for directory structure and error handling.
func storePageHTML_Placeholder(htmlContent string, subForum data.SubForum, topic data.Topic, pageNum int, pageURL string, cfg *config.Config) error {
	// Determine file path based on config and conventions (platform-agnostic)
	// Example: <ArchiveOutputRootDir>/<SubForumID>/<TopicID>/page_<PageNum>.html
	topicDir := filepath.Join(cfg.ArchiveOutputRootDir, strconv.Itoa(subForum.ID), topic.ID)
	fileName := fmt.Sprintf("page_%d.html", pageNum)
	fullPath := filepath.Join(topicDir, fileName)

	log.Printf("[INFO] STORAGE_PLACEHOLDER: Attempting to store HTML for Topic ID: %s, Page: %d (URL: %s) to '%s'",
		topic.ID, pageNum, pageURL, fullPath)

	// Simulate a storage error for testing
	if topic.ID == "topic_with_storage_error" && pageNum == 1 {
		log.Printf("[ERROR] STORAGE_PLACEHOLDER: Simulated storage error for %s", fullPath)
		return fmt.Errorf("simulated storage error for %s", fullPath)
	}

	// In a real implementation, this would involve:
	// 1. os.MkdirAll(topicDir, 0755) to ensure directory exists.
	// 2. os.WriteFile(fullPath, []byte(htmlContent), 0644) to save the content.

	log.Printf("[INFO] STORAGE_PLACEHOLDER: Successfully stored HTML to %s. Content length: %d bytes.", fullPath, len(htmlContent))
	return nil
}

// main is the entry point for the Waypoint Archiver.
// It orchestrates the archival process: configuration, state loading, topic indexing,
// JIT refresh, iterating through sub-forums and topics, and processing individual pages.
func main() {
	log.Println("[DEBUG] main: Script started.")
	log.Printf("[DEBUG] main: os.Args = %v", os.Args)
	// --- Configuration Loading ---
	cfg, err := config.LoadConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			log.Println("[INFO] Help flag detected or error led to help display. Exiting cleanly.")
			os.Exit(0)
		}
		log.Fatalf("[FATAL] Failed to load configuration: %v", err)
	}
	log.Println("[DEBUG] main: Configuration loaded successfully.")

	logFile := initLogging(cfg)
	if logFile != nil {
		defer func() {
			log.Printf("[DEBUG] main: Defer: Attempting to close logFile from main: %s", cfg.LogFilePath)
			errClose := logFile.Close()
			if errClose != nil {
				fmt.Fprintf(os.Stderr, "[CRITICAL] main: Defer: Failed to close log file %s: %v\n", cfg.LogFilePath, errClose)
			}
		}()
	}
	log.Println("[DEBUG] main: initLogging completed.")

	metrics.InitPerformanceLogger(cfg.PerformanceLogPath)
	log.Println("[DEBUG] main: InitPerformanceLogger completed.")

	// Determine currentArchiveRoot BEFORE initializing storer
	currentArchiveRoot := cfg.ArchiveOutputRootDir // Default, might be overridden by test config
	log.Printf("[DEBUG] main: Initial currentArchiveRoot set from cfg.ArchiveOutputRootDir: %s", currentArchiveRoot)

	if len(cfg.TestSubForumIDs) > 0 {
		log.Printf("[INFO] TEST MODE: TestSubForumIDs specified (%v). Using TestArchiveOutputRoot: %s", cfg.TestSubForumIDs, cfg.TestArchiveOutputRoot)
		currentArchiveRoot = cfg.TestArchiveOutputRoot
		if err := os.MkdirAll(currentArchiveRoot, 0755); err != nil {
			log.Fatalf("[FATAL] Failed to create test archive output directory %s: %v", currentArchiveRoot, err)
		}
	} else {
		// If not in test mode, ensure the production archive root exists
		log.Printf("[INFO] PRODUCTION MODE: TestSubForumIDs not specified or empty. Using ArchiveOutputRootDir: %s", cfg.ArchiveOutputRootDir)
		// currentArchiveRoot is already cfg.ArchiveOutputRootDir in this case
		if err := os.MkdirAll(currentArchiveRoot, 0755); err != nil {
			log.Fatalf("[FATAL] Failed to create archive output directory %s: %v", currentArchiveRoot, err)
		}
	}
	log.Printf("[DEBUG] main: Final currentArchiveRoot for storer: %s", currentArchiveRoot)

	htmlStorer := storer.NewStorer(currentArchiveRoot) // Pass the correct root
	log.Println("[DEBUG] main: htmlStorer created.")

	// Create instances for JIT refresh dependencies
	htmlFetcher := htmlutil.NewHTMLFetcher(cfg.UserAgent, cfg.PolitenessDelay)
	log.Println("[DEBUG] main: htmlFetcher created.")
	htmlParser := htmlutil.NewPaginationParser(cfg.ForumBaseURL)
	log.Println("[DEBUG] main: htmlParser created.")
	htmlExtractor := htmlutil.NewTopicExtractor(cfg.ForumBaseURL)
	log.Println("[DEBUG] main: htmlExtractor created.")

	pageDownloader := downloader.NewDownloader(cfg)
	log.Println("[DEBUG] main: pageDownloader created.")
	currentBatchMetrics := metrics.NewBatchMetrics()
	log.Println("[DEBUG] main: currentBatchMetrics created.")
	lastMetricsSave := time.Now() // Keep for periodic detail metrics save
	log.Println("[DEBUG] main: lastMetricsSave initialized.")

	// --- Topic Index Loading ---
	log.Printf("[INFO] Loading sub-forum list from: %s", cfg.SubForumListFile)
	// Changed from ReadSubForumListCSV to ReadSubForumListJSON
	initialSubForums, err := indexerlogic.ReadSubForumListJSON(cfg.SubForumListFile)
	if err != nil {
		log.Fatalf("[FATAL] Failed to read sub-forum list %s: %v", cfg.SubForumListFile, err)
	}
	log.Printf("[DEBUG] main: SubForumListJSON read, %d entries.", len(initialSubForums))

	var allTopicsMasterList []data.Topic
	var allSubForumsList []data.SubForum

	// Loop changed to iterate over initialSubForums (slice of data.SubForum)
	for _, sfBase := range initialSubForums {
		// Construct path to specific topic index JSON
		// Assumes TopicIndexDir is like ".../indexed_data/"
		// And TopicIndexFilePattern is like "topic_index_*.json"
		// And actual files are in ".../indexed_data/forum_XX/topic_index_XX.json"
		forumDirComponent := "forum_" + strconv.Itoa(sfBase.ID)
		// Use sfBase.ID for the wildcard replacement in the pattern
		topicIndexFilename := strings.Replace(cfg.TopicIndexFilePattern, "*", strconv.Itoa(sfBase.ID), 1)
		topicDataPath := filepath.Join(cfg.TopicIndexDir, forumDirComponent, topicIndexFilename)

		log.Printf("[DEBUG] Loading topics for SubForum ID: %d (Name: %s) from %s", sfBase.ID, sfBase.Name, topicDataPath)
		topicsForThisSF, err := indexerlogic.ReadTopicIndexJSON(topicDataPath, strconv.Itoa(sfBase.ID))
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("[WARNING] Topic index file %s not found for sub-forum %d (Name: %s). Treating as 0 topics initially.", topicDataPath, sfBase.ID, sfBase.Name)
			} else {
				log.Printf("[WARNING] Failed to read topic index for sub-forum %d (Name: %s) from %s: %v. Treating as 0 topics initially.", sfBase.ID, sfBase.Name, topicDataPath, err)
			}
			// Create a SubForum entry even if topics couldn't be loaded, so it's in the list for potential JIT
			allSubForumsList = append(allSubForumsList, data.SubForum{
				ID:         sfBase.ID,
				Name:       sfBase.Name,
				URL:        sfBase.URL,
				TopicCount: sfBase.TopicCount, // Use count from subforum_list.json initially
				Topics:     make([]data.Topic, 0),
			})
			continue
		}

		allTopicsMasterList = append(allTopicsMasterList, topicsForThisSF...)
		subForumEntry := data.SubForum{
			ID:         sfBase.ID,
			Name:       sfBase.Name,
			URL:        sfBase.URL,
			TopicCount: len(topicsForThisSF), // Actual count of loaded topics
			Topics:     topicsForThisSF,
		}
		allSubForumsList = append(allSubForumsList, subForumEntry)
		log.Printf("[DEBUG] Successfully loaded %d topics for SubForum ID: %d.", len(topicsForThisSF), sfBase.ID)
	}
	masterTopicList := &data.MasterTopicList{Topics: allTopicsMasterList}
	log.Printf("[INFO] Successfully processed %d sub-forums, loaded %d unique topics into the master list.", len(allSubForumsList), len(masterTopicList.Topics))

	// Filter and sort sub-forums for processing
	var sortedSubForums []data.SubForum
	if len(cfg.TestSubForumIDs) > 0 {
		sortedSubForums = make([]data.SubForum, 0, len(allSubForumsList))
		for _, sf := range allSubForumsList {
			if len(cfg.TestSubForumIDs) > 0 {
				found := false
				for _, testID := range cfg.TestSubForumIDs {
					if strconv.Itoa(sf.ID) == testID {
						found = true
						break
					}
				}
				if !found {
					continue
				}
			}
			sortedSubForums = append(sortedSubForums, sf)
		}
		log.Printf("[INFO] Found %d sub-forums to process after filtering.", len(sortedSubForums))
	} else {
		sortedSubForums = allSubForumsList
	}

	// Load existing state if available
	archivalState, err := state.LoadState(cfg.StateFilePath)
	if err != nil {
		log.Printf("[WARNING] Could not load existing state from %s (will start fresh): %v", cfg.StateFilePath, err)
		archivalState = state.NewArchiveProgressState() // Corrected: Use NewArchiveProgressState
	}
	// Ensure maps are initialized if they were nil in the JSON (e.g. empty file or old format)
	// This is handled by LoadState and NewArchiveProgressState now.
	log.Printf("[INFO] Initial state loaded. %d topics marked as archived.", len(archivalState.ArchivedTopics)) // Corrected: ArchivedTopics, removed TopicErrors

	// Calculate total topics for progress tracking (only for selected sub-forums)
	totalTopicsOverallForRun := 0
	for _, sf := range sortedSubForums {
		totalTopicsOverallForRun += sf.TopicCount
	}
	processedTopicsSoFar := 0 // Tracks topics processed across all subforums in this run for ETC

	// --- Main Archival Loop ---
	log.Println("[INFO] Starting main archival loop...")
	// startTime := time.Now() // Removed, use currentBatchMetrics.StartTime

	// Setup context for graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Resume logic variables
	skipToSubForumID := archivalState.LastProcessedSubForumID
	skipToTopicID := archivalState.LastProcessedTopicID
	startPageForTopic := archivalState.LastProcessedPageNumberInTopic + 1 // if > 0, means topic was partial
	foundResumedSubForum := false

	for _, currentSubForum := range sortedSubForums {
		sfProcessStartTime := time.Now()
		pagesArchivedInSubForum := 0

		// Sub-forum level resume logic
		if skipToSubForumID != "" && strconv.Itoa(currentSubForum.ID) != skipToSubForumID && !foundResumedSubForum {
			log.Printf("[INFO] Resume: Skipping SubForum ID %d (waiting for resume point %s).", currentSubForum.ID, skipToSubForumID)
			processedTopicsSoFar += currentSubForum.TopicCount // Account for skipped topics in ETC
			continue
		}
		if strconv.Itoa(currentSubForum.ID) == skipToSubForumID {
			foundResumedSubForum = true
		}
		// If no specific sub-forum to skip to, or if we've found it, proceed.

		// Check if this sub-forum is in the test list (if test mode is active)
		if len(cfg.TestSubForumIDs) > 0 {
			foundInTestList := false
			for _, testID := range cfg.TestSubForumIDs {
				if strconv.Itoa(currentSubForum.ID) == testID {
					foundInTestList = true
					break
				}
			}
			if !foundInTestList {
				log.Printf("[DEBUG] TEST MODE: Skipping sub-forum %d (%s) as it's not in TestSubForumIDs list.", currentSubForum.ID, currentSubForum.Name)
				continue
			}
		}

		log.Printf("[INFO] >>> Processing Sub-Forum ID: %d, Name: %s, URL: %s", currentSubForum.ID, currentSubForum.Name, currentSubForum.URL)

		// --- JIT Topic Index Refresh (Story 2.8 AC9) ---
		topicsForSubForum := currentSubForum.Topics // Work with a copy that can be modified by JIT

		// Conditions for attempting JIT refresh:
		if cfg.JITRefreshPages > 0 && currentSubForum.URL != "" {
			// Assuming cfg.JITRefreshInterval is in minutes for this conversion
			jitIntervalDuration := time.Duration(cfg.JITRefreshInterval) * time.Minute
			// ShouldPerformJITRefresh handles its own logic based on state, including checking the interval.
			if jitrefresh.ShouldPerformJITRefresh(currentSubForum, archivalState, cfg.JITRefreshPages > 0, jitIntervalDuration) {
				log.Printf("[INFO] JITREFRESH: Performing JIT index refresh for SubForum %d (URL: %s, first %d pages)", currentSubForum.ID, currentSubForum.URL, cfg.JITRefreshPages)
				// Corrected JIT Call: Pass instances, not methods
				newlyDiscoveredTopics, errJIT := jitrefresh.PerformJITRefresh(
					currentSubForum, // Pass the current subForum from the loop
					cfg,
					htmlFetcher,   // Pass the instance
					htmlParser,    // Pass the instance
					htmlExtractor, // Pass the instance
				)

				if errJIT != nil {
					log.Printf("[WARNING] JITREFRESH: Error during JIT refresh for %d: %v. Proceeding with initially indexed topics.", currentSubForum.ID, errJIT)
				} else if len(newlyDiscoveredTopics) > 0 {
					log.Printf("[INFO] JITREFRESH: Found %d new/updated topics for %d.", len(newlyDiscoveredTopics), currentSubForum.ID)
					// Merge logic: Add new topics, potentially update existing ones. For now, just append and let processing handle duplicates if any based on ID.
					existingTopicMap := make(map[string]bool)
					for _, t := range topicsForSubForum {
						existingTopicMap[t.ID] = true
					}
					addedCount := 0
					for _, newTopic := range newlyDiscoveredTopics {
						if !existingTopicMap[newTopic.ID] {
							topicsForSubForum = append(topicsForSubForum, newTopic)
							addedCount++
						}
					}
					if addedCount > 0 {
						log.Printf("[INFO] JITREFRESH: Added %d unique new topics to process for subforum %d.", addedCount, currentSubForum.ID)
						totalTopicsOverallForRun += addedCount // Adjust total for ETC
						// Re-sort if order matters after JIT additions
						sort.Slice(topicsForSubForum, func(i, j int) bool {
							return topicsForSubForum[i].ID < topicsForSubForum[j].ID
						})
					}
				}
			}
		} else {
			log.Printf("[INFO] JIT REFRESH: Sub-forum %d has no base URL. Cannot perform JIT refresh. Using %d loaded topics.", currentSubForum.ID, len(topicsForSubForum))
		}

		// Topic-level resume: Determine if we need to skip topics within this sub-forum
		foundResumedTopicInSF := false
		if strconv.Itoa(currentSubForum.ID) != skipToSubForumID || skipToTopicID == "" {
			// If this isn't the sub-forum we stopped on, or if there was no specific topic, process all topics in this SF
			foundResumedTopicInSF = true
		}

		if len(topicsForSubForum) == 0 {
			log.Printf("[INFO] No topics found or loaded for sub-forum %d. Moving to next sub-forum.", currentSubForum.ID)
			continue
		}

		// --- Topic Loop for the current Sub-Forum ---
		log.Printf("[INFO] Starting topic loop for sub-forum %d (%d topics)...", currentSubForum.ID, len(topicsForSubForum))
		for topicIndex, topic := range topicsForSubForum { // topic is already data.Topic
			// Resume logic for topics
			if !foundResumedTopicInSF && topic.ID != skipToTopicID {
				log.Printf("[INFO] Resume: SF %d: Skipping Topic ID %s (waiting for resume topic %s).", currentSubForum.ID, topic.ID, skipToTopicID)
				processedTopicsSoFar++ // Account for ETC
				continue
			}
			if strconv.Itoa(currentSubForum.ID) == skipToSubForumID && topic.ID == skipToTopicID {
				foundResumedTopicInSF = true // This is the specific topic to start/resume from
			}

			// Ensure SubForumID is set correctly on the topic (JIT refresh should handle this)
			// topic.SubForumID = currentSubForum.ID // Redundant if JIT refresh sets it

			select {
			case <-ctx.Done():
				log.Println("[INFO] ARCHIVER: Shutdown signal received. Saving state and exiting...")
				state.SaveProgress(cfg.StateFilePath)
				metrics.SaveDetailMetricsLog() // Save metrics on graceful shutdown too
				return
			default:
			}

			log.Printf("[PROGRESS] Sub-forum %d (%s): Processing topic %d/%d (ID: %s, Title: %s)", currentSubForum.ID, currentSubForum.Name, topicIndex+1, len(topicsForSubForum), topic.ID, topic.Title)

			// Check if topic already completed or has a persistent error
			if archivalState.IsTopicArchived(topic.ID) {
				log.Printf("[INFO] ARCHIVAL: Topic %s is already archived. Skipping.", topic.ID)
				continue // Next topic
			}
			topicStartTime := time.Now() // For timing individual topic processing

			// --- Page Loop for the current Topic ---
			// pagesInTopic := 0 // Removed, use pagesProcessedThisRunForTopic
			pagesProcessedThisRunForTopic := 0

			// For each topic, discover all its pages - This logic is from the user's "clean slate"
			topicPageURLs := make([]string, 0, 5) // Pre-allocate a bit
			allTopicPages := make(map[string]bool)
			pagesToScan := make([]string, 0, 5)

			normalizedInitialURL, errNormInitial := util.NormalizeTopicPageURL(topic.URL, topic.SubForumID)
			if errNormInitial != nil {
				log.Printf("[ERROR] NAV: Failed to normalize initial topic URL '%s' for topic %s: %v. Skipping this topic.", topic.URL, topic.ID, errNormInitial)
				currentBatchMetrics.ErrorsEncountered++
				// archivalState.RecordTopicError(topic.ID, fmt.Sprintf("Failed to normalize initial URL: %v", errNormInitial)) // Removed
				continue // to the next topic
			}
			topicPageURLs = append(topicPageURLs, normalizedInitialURL)
			allTopicPages[normalizedInitialURL] = true
			pagesToScan = append(pagesToScan, normalizedInitialURL)
			log.Printf("[DEBUG] NAV: Added initial normalized page %s to scan queue for topic %s", normalizedInitialURL, topic.ID)

			for len(pagesToScan) > 0 {
				currentScanURL := pagesToScan[0]
				pagesToScan = pagesToScan[1:]

				log.Printf("[DEBUG] NAV: Fetching %s for pagination (topic %s)", currentScanURL, topic.ID)
				pageHTML, errFetch := htmlFetcher.FetchHTML(currentScanURL)
				if errFetch != nil {
					log.Printf("[ERROR] NAV: Error fetching %s for pagination: %v", currentScanURL, errFetch)
					currentBatchMetrics.ErrorsEncountered++
					// archivalState.RecordTopicError(topic.ID, fmt.Sprintf("NAV: Error fetching %s for pagination: %v", currentScanURL, errFetch)) // Removed
					continue // Skip this page if fetch fails, but attempt to process other pages of the topic
				}

				log.Printf("[DEBUG] NAV: Parsing %s for pagination links (topic %s)", currentScanURL, topic.ID)
				paginationLinks, errParse := htmlParser.ParsePaginationLinks(pageHTML, currentScanURL)
				if errParse != nil {
					log.Printf("[ERROR] NAV: Error parsing pagination links from %s: %v", currentScanURL, errParse)
					currentBatchMetrics.ErrorsEncountered++
					// archivalState.RecordTopicError(topic.ID, fmt.Sprintf("NAV: Error parsing pagination links from %s: %v", currentScanURL, errParse)) // Removed
					continue // Skip if parsing fails
				}

				for _, link := range paginationLinks {
					normalizedLink, errNorm := util.NormalizeTopicPageURL(link, topic.SubForumID)
					if errNorm != nil {
						log.Printf("[WARNING] NAV: Failed to normalize pagination link '%s': %v. Skipping.", link, errNorm)
						continue
					}

					if !allTopicPages[normalizedLink] {
						allTopicPages[normalizedLink] = true
						topicPageURLs = append(topicPageURLs, normalizedLink)
						pagesToScan = append(pagesToScan, normalizedLink)
						log.Printf("[DEBUG] NAV: Added unique normalized page %s to scan queue for topic %s", normalizedLink, topic.ID)
					}
				}
			}
			log.Printf("[INFO] NAV: Found %d unique pages for Topic ID %s.", len(topicPageURLs), topic.ID)
			// End of "clean slate" page discovery logic

			for pageNum0Based, pageURL := range topicPageURLs { // Iterate using topicPageURLs
				actualPageNum := pageNum0Based + 1 // 1-based for logging and storage
				pageID := fmt.Sprintf("%s_p%d", topic.ID, actualPageNum)

				// Resume logic for pages within a topic
				if strconv.Itoa(currentSubForum.ID) == skipToSubForumID &&
					topic.ID == skipToTopicID &&
					actualPageNum < startPageForTopic {
					log.Printf("[INFO] Resume: SF %d, Topic %s: Skipping already processed page %d (resuming from page %d).", currentSubForum.ID, topic.ID, actualPageNum, startPageForTopic)
					continue
				}

				log.Printf("[INFO] Archiving page %d for topic %s (URL: %s)", actualPageNum, topic.ID, pageURL)
				pageProcessStartTime := time.Now()

				// Download page HTML
				htmlContentBytes, err := pageDownloader.FetchPage(pageURL)
				if err != nil {
					log.Printf("[ERROR] DOWNLOAD: Failed to download page %s for topic %s: %v", pageURL, topic.ID, err)
					// archivalState.RecordTopicError(topic.ID, fmt.Sprintf("Failed to download page %s: %v", pageURL, err)) // Removed
					currentBatchMetrics.ErrorsEncountered++
					// topicFailed = true // Removed
					// Decide: break from page loop for this topic, or try next page?
					// For now, if one page fails, we log error, count it, and continue to next page of THIS topic.
					// If all pages of a topic fail, the topic itself won't be marked as "Archived" in the state.
					metrics.AppendDetailMetric(metrics.PerformanceMetric{Timestamp: time.Now(), ResourceType: metrics.ResourceTypeTopicPage, ResourceID: pageID, Action: metrics.ActionSkipped, Size: 0, Duration: time.Since(pageProcessStartTime), Notes: fmt.Sprintf("download error: %v", err)})
					continue // Continue to the next page of the current topic
				}

				// Store page HTML
				savedPath, err := storePageHTML(htmlStorer, topic, strconv.Itoa(currentSubForum.ID), actualPageNum, htmlContentBytes) // Converted currentSubForum.ID
				if err != nil {
					log.Printf("[ERROR] STORAGE: Failed to store page %s for topic %s: %v", pageURL, topic.ID, err)
					// archivalState.RecordTopicError(topic.ID, fmt.Sprintf("Failed to store page %s: %v", pageURL, err)) // Removed
					currentBatchMetrics.ErrorsEncountered++
					// topicFailed = true // Removed
					metrics.AppendDetailMetric(metrics.PerformanceMetric{Timestamp: time.Now(), ResourceType: metrics.ResourceTypeTopicPage, ResourceID: pageID, Action: metrics.ActionSkipped, Size: 0, Duration: time.Since(pageProcessStartTime), Notes: fmt.Sprintf("storage error: %v", err)})
					continue // Continue to the next page of the current topic
				}
				log.Printf("[INFO] ARCHIVER: Saved HTML for topic %s, page %d to %s", topic.ID, actualPageNum, savedPath)
				currentBatchMetrics.PagesArchived++
				currentBatchMetrics.BytesArchived += int64(len(htmlContentBytes))
				pagesArchivedInSubForum++                                          // Increment for sub-forum summary
				pagesProcessedThisRunForTopic++                                    // Increment for topic summary
				archivalState.MarkPageAsArchived(topic.ID, actualPageNum, pageURL) // Mark page in state
				archivalState.LastProcessedPageNumberInTopic = actualPageNum       // Update for resume
				metrics.AppendDetailMetric(metrics.PerformanceMetric{Timestamp: time.Now(), ResourceType: metrics.ResourceTypeTopicPage, ResourceID: pageID, Action: metrics.ActionArchived, Size: int64(len(htmlContentBytes)), Duration: time.Since(pageProcessStartTime)})
			} // End page loop

			if pagesProcessedThisRunForTopic == len(topicPageURLs) && len(topicPageURLs) > 0 { // All pages fetched and stored successfully
				archivalState.MarkTopicAsArchived(topic.ID)
				log.Printf("[INFO] ARCHIVER: Topic %s marked as archived. Pages processed in this run: %d. Total duration for topic: %s", topic.ID, pagesProcessedThisRunForTopic, time.Since(topicStartTime).Round(time.Second))
				currentBatchMetrics.TopicsArchived++
				processedTopicsSoFar++ // For ETC
			} else if len(topicPageURLs) == 0 {
				log.Printf("[INFO] ARCHIVER: Topic %s has no pages to archive (or URL was invalid). Skipping topic archival marking.", topic.ID)
			} else {
				log.Printf("[WARNING] ARCHIVER: Topic %s completed processing, but only %d out of %d pages were successfully archived. Topic not marked as fully archived.", topic.ID, pagesProcessedThisRunForTopic, len(topicPageURLs))
			}

			archivalState.LastProcessedTopicID = topic.ID
			archivalState.LastProcessedPageNumberInTopic = 0 // Reset for the next topic

			// Save state based on interval
			// TODO: Review if SaveStateInterval should be an int (number of topics) in config instead of time.Duration
			saveIntervalInSeconds := int(cfg.SaveStateInterval.Seconds())
			if saveIntervalInSeconds <= 0 || (topicIndex+1)%saveIntervalInSeconds == 0 { // Use topicIndex for periodic save
				log.Printf("[DEBUG] ARCHIVER: Saving state post-topic %s (interval: %d topics, current index: %d).", topic.ID, saveIntervalInSeconds, topicIndex+1)
				state.SaveProgress(cfg.StateFilePath)
			}
			currentBatchMetrics.UpdateRates()
			displayProgressAndETC(processedTopicsSoFar, totalTopicsOverallForRun, currentBatchMetrics) // Update and display progress

			if time.Since(lastMetricsSave) > 5*time.Minute { // Check if lastMetricsSave is used
				log.Println("[INFO] ARCHIVER: Periodically saving detailed performance metrics...")
				if err := metrics.SaveDetailMetricsLog(); err != nil { // This function's existence is assumed
					log.Printf("[ERROR] ARCHIVER: Periodic metrics save failed: %v", err)
				}
				lastMetricsSave = time.Now()
			}
		} // End topic loop

		archivalState.LastProcessedSubForumID = strconv.Itoa(currentSubForum.ID) // Converted currentSubForum.ID
		archivalState.ProcessedTopicIDsInCurrentSubForum = []string{}            // Clear for next sub-forum
		allTopicsInSFProcessed := true
		for _, t := range topicsForSubForum {
			if !archivalState.IsTopicArchived(t.ID) {
				allTopicsInSFProcessed = false
				break
			}
		}
		if allTopicsInSFProcessed && len(topicsForSubForum) > 0 {
			// Check if already completed to avoid duplicates, before trying to use util.UniqueStrings
			alreadyMarkedCompleted := false
			for _, completedID := range archivalState.CompletedSubForumIDs {
				if completedID == strconv.Itoa(currentSubForum.ID) { // Converted currentSubForum.ID
					alreadyMarkedCompleted = true
					break
				}
			}
			if !alreadyMarkedCompleted {
				archivalState.CompletedSubForumIDs = append(archivalState.CompletedSubForumIDs, strconv.Itoa(currentSubForum.ID)) // Converted currentSubForum.ID
				// TODO: Implement or provide util.UniqueStrings if needed, or ensure this append logic is sufficient.
				archivalState.CompletedSubForumIDs = uniqueStringsSlice(archivalState.CompletedSubForumIDs) // Use local helper
			}
		}

		log.Printf("[INFO] <<< Finished processing Sub-Forum ID: %d. Duration: %s. Topics in SF: %d, Pages Archived in this SF: %d",
			currentSubForum.ID, time.Since(sfProcessStartTime).Round(time.Second), len(topicsForSubForum), pagesArchivedInSubForum) // Used pagesArchivedInSubForum
		// metrics.LogBatchMetrics(currentBatchMetrics, cfg.SubForumListFile) // TODO: Implement historical/summary metric logging for sub-forum if needed
		// currentBatchMetrics = metrics.NewBatchMetrics() // Reset for next sub-forum OR do a single batch metric for the whole run.
		// For now, currentBatchMetrics accumulates for the whole run.
	} // End of sub-forum loop

	log.Println("[INFO] Archival process completed.")

	// Final state save
	log.Println("[INFO] ARCHIVER: Final state save before exiting...")
	state.SaveProgress(cfg.StateFilePath)

	// Save all buffered performance metrics
	log.Println("[INFO] ARCHIVER: Saving all performance metrics to CSV...")
	if err := metrics.SaveDetailMetricsLog(); err != nil {
		log.Printf("[ERROR] ARCHIVER: Failed to save final performance metrics: %v", err)
	} else {
		log.Println("[INFO] ARCHIVER: Final performance metrics saved to CSV.")
	}

	// Log final summary metrics for the entire run
	finalMetricsSummary := currentBatchMetrics.ToHistoricalMetrics("overall_run_summary")
	elapsedRunTime := time.Since(currentBatchMetrics.StartTime)
	log.Printf("[INFO] ARCHIVER: === Run Summary ===")
	log.Printf("[INFO] ARCHIVER: Total Run Time: %s", metrics.FormatDuration(elapsedRunTime))
	log.Printf("[INFO] ARCHIVER: Total Topics Processed: %d", finalMetricsSummary.TopicsArchived)
	log.Printf("[INFO] ARCHIVER: Total Pages Processed: %d", finalMetricsSummary.PagesArchived)
	log.Printf("[INFO] ARCHIVER: Total Bytes Archived: %s", metrics.FormatBytes(finalMetricsSummary.BytesArchived))
	log.Printf("[INFO] ARCHIVER: Average Topics/Hour: %.2f", finalMetricsSummary.AvgTopicsPerHour)
	log.Printf("[INFO] ARCHIVER: Average Pages/Min: %.2f", finalMetricsSummary.AvgPagesPerMin)
	log.Printf("[INFO] ARCHIVER: Average MB/Min: %.2f", finalMetricsSummary.AvgMBPerMin)
	log.Printf("[INFO] ARCHIVER: ===================")

	log.Println("[INFO] ARCHIVER: Archival run finished.") // True final operational message
}

// initLogging configures the global logger and returns the log file if successful.
func initLogging(cfg *config.Config) *os.File {
	var logFileHandle *os.File = nil
	if cfg.LogFilePath != "" {
		logDir := filepath.Dir(cfg.LogFilePath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("[WARNING] initLogging: Failed to create log directory %s: %v. Logging to stdout/stderr.", logDir, err)
		} else {
			var errOpenFile error
			logFileHandle, errOpenFile = os.OpenFile(cfg.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if errOpenFile != nil {
				log.Printf("[WARNING] initLogging: Failed to open log file %s: %v. Logging to stdout/stderr.", cfg.LogFilePath, errOpenFile)
				logFileHandle = nil // Ensure it's nil if open failed
			} else {
				// DO NOT DEFER CLOSE HERE. main will handle it.
				log.SetOutput(logFileHandle)
				log.Printf("[INFO] initLogging: Successfully redirected log output to file: %s", cfg.LogFilePath)
			}
		}
	}

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Printf("[INFO] Log level set to: %s (Note: Standard logger does not filter by level)", cfg.LogLevel)
	return logFileHandle
}

// storePageHTML stores the provided HTML content to the appropriate file path.
func storePageHTML(s *storer.Storer, topic data.Topic, subForumID string, pageNum int, htmlBytes []byte) (string, error) {
	filePath, err := s.SaveTopicHTML(subForumID, topic.ID, pageNum, htmlBytes)
	if err != nil {
		return "", fmt.Errorf("failed to save HTML for topic %s, page %d: %w", topic.ID, pageNum, err)
	}
	return filePath, nil
}

// displayProgressAndETC calculates and displays the current progress and estimated time to completion.
func displayProgressAndETC(processedTopics, totalTopics int, bm *metrics.BatchMetrics) {
	remainingTopics := int64(totalTopics - processedTopics)
	// For page-based ETC, we'd need a good estimate of remaining pages.
	// For now, ETC primarily based on topics.
	estimatedETC := bm.GetETC(0, remainingTopics) // Pass 0 for remainingPages if not accurately tracked yet

	progressMsg := metrics.FormatProgress(int64(processedTopics), int64(totalTopics))
	ratesMsg := metrics.FormatRates(bm.CurrentPagesPerMin, bm.CurrentTopicsPerHour, bm.CurrentMBPerMin)
	etcMsg := metrics.FormatETC(estimatedETC)

	log.Printf("[PROGRESS] %s | %s | %s | Total Runtime: %s",
		progressMsg,
		ratesMsg,
		etcMsg,
		metrics.FormatDuration(time.Since(bm.StartTime)),
	)
}

// getAllPageURLsForTopic is no longer used directly here, logic integrated into main loop.
// If it were to be kept separate, its signature would need to match:
// func getAllPageURLsForTopic(topic data.Topic, cfg *config.Config, fetcher func(string, time.Duration, string) (string, error), parser func(string, string) ([]string, error)) ([]string, error) { ... }

// main entry point
// func main() {
// runArchiver()
// }

// For reference, if runArchiver were to return an error for testing or other purposes:
// func runArchiver() error { ... return nil / fmt.Errorf(...) ... }
// For now, main calls runArchiver which os.Exit or log.Fatalf on critical errors.

// uniqueStringsSlice returns a new slice with only unique strings from the input slice.
func uniqueStringsSlice(stringSlice []string) []string {
	allKeys := make(map[string]bool)
	list := []string{}
	for _, item := range stringSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
