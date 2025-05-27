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
	topicDir := filepath.Join(cfg.ArchiveOutputRootDir, subForum.ID, topic.ID)
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
	lastMetricsSave := time.Now()
	log.Println("[DEBUG] main: lastMetricsSave initialized.")

	// --- Topic Index Loading ---
	log.Printf("[INFO] Loading sub-forum list from: %s", cfg.SubForumListFile)
	subForumDetailsMap, err := indexerlogic.ReadSubForumListCSV(cfg.SubForumListFile)
	if err != nil {
		log.Fatalf("[FATAL] Failed to read sub-forum list %s: %v", cfg.SubForumListFile, err)
	}
	log.Printf("[DEBUG] main: SubForumListCSV read, %d entries.", len(subForumDetailsMap))

	var allTopicsMasterList []data.Topic
	var allSubForumsList []data.SubForum

	for sfID, sfDetails := range subForumDetailsMap {
		// Construct path to specific topic index JSON using the pattern from config
		// Example: cfg.TopicIndexDir = "../master_output/indexed_data"
		//          cfg.TopicIndexFilePattern = "forum_%s/topic_index_%s.json"
		//          sfID = "39"
		// Results in: "../master_output/indexed_data/forum_39/topic_index_39.json"
		topicIndexRelativePath := fmt.Sprintf(cfg.TopicIndexFilePattern, sfID, sfID)
		topicDataPath := filepath.Join(cfg.TopicIndexDir, topicIndexRelativePath)

		topicsForThisSF, err := indexerlogic.ReadTopicIndexJSON(topicDataPath, sfID)
		if err != nil {
			if os.IsNotExist(err) {
				log.Printf("[WARNING] Topic index file %s not found for sub-forum %s. Skipping sub-forum.", topicDataPath, sfID)
			} else {
				log.Printf("[WARNING] Failed to read topic index for sub-forum %s from %s: %v. Skipping sub-forum.", sfID, topicDataPath, err)
			}
			// Create a SubForum entry even if topics couldn't be loaded, so it's in the list for potential JIT
			allSubForumsList = append(allSubForumsList, data.SubForum{
				ID:         sfID,
				Name:       sfDetails.Name,
				URL:        sfDetails.URL,
				TopicCount: 0,
				Topics:     make([]data.Topic, 0),
			})
			continue
		}

		allTopicsMasterList = append(allTopicsMasterList, topicsForThisSF...)
		subForumEntry := data.SubForum{
			ID:         sfID,
			Name:       sfDetails.Name,
			URL:        sfDetails.URL,
			TopicCount: len(topicsForThisSF),
			Topics:     topicsForThisSF, // Store the actual topics with the subforum
		}
		allSubForumsList = append(allSubForumsList, subForumEntry)
	}
	masterTopicList := &data.MasterTopicList{Topics: allTopicsMasterList} // Not directly used further if subforums hold their topics
	log.Printf("[INFO] Successfully processed %d sub-forums, loaded %d unique topics into the master list.", len(allSubForumsList), len(masterTopicList.Topics))

	// Filter and sort sub-forums for processing
	var sortedSubForums []data.SubForum
	if len(cfg.TestSubForumIDs) > 0 {
		log.Printf("[INFO] TEST MODE: Filtering sub-forums for IDs: %v", cfg.TestSubForumIDs)
		// Explicitly order for 39 then 105 if they are in TestSubForumIDs and allSubForumsList
		foundSFs := make(map[string]data.SubForum)
		for _, sf := range allSubForumsList {
			foundSFs[sf.ID] = sf
		}
		for _, testID := range cfg.TestSubForumIDs { // Iterate in the order of TestSubForumIDs ("39", then "105")
			if sf, ok := foundSFs[testID]; ok {
				sortedSubForums = append(sortedSubForums, sf)
			}
		}
	} else {
		sortedSubForums = allSubForumsList
		// Sort all if not in test mode
		sort.Slice(sortedSubForums, func(i, j int) bool {
			return sortedSubForums[i].ID < sortedSubForums[j].ID
		})
	}

	if len(sortedSubForums) == 0 {
		log.Fatalf("[FATAL] No sub-forums selected for processing. If in test mode, check TestSubForumIDs and TopicIndexDir contents/naming. Exiting.")
	}
	log.Printf("[INFO] Found %d sub-forums to process after filtering.", len(sortedSubForums))

	// --- State Loading & Initialization ---
	log.Printf("[INFO] Loading archival progress state from: %s", cfg.StateFilePath)
	loadedState, err := state.LoadState(cfg.StateFilePath)
	if err != nil {
		log.Fatalf("[FATAL] Error loading archival state from %s: %v", cfg.StateFilePath, err)
	}
	if loadedState == nil {
		log.Println("[INFO] No previous state file found or file was empty. Starting fresh archival.")
		state.CurrentState = &state.ArchiveProgressState{
			ProcessedTopicIDsInCurrentSubForum: make([]string, 0),
			CompletedSubForumIDs:               make([]string, 0),
		}
	} else {
		log.Println("[INFO] Successfully loaded previous archival state.")
		state.CurrentState = loadedState
		log.Printf("[INFO] Resuming from: SubForumID: %s, TopicID: %s, Page: %d",
			state.CurrentState.LastProcessedSubForumID,
			state.CurrentState.LastProcessedTopicID,
			state.CurrentState.LastProcessedPageNumberInTopic)
	}
	if state.CurrentState.ProcessedTopicIDsInCurrentSubForum == nil {
		state.CurrentState.ProcessedTopicIDsInCurrentSubForum = make([]string, 0)
	}
	if state.CurrentState.CompletedSubForumIDs == nil {
		state.CurrentState.CompletedSubForumIDs = make([]string, 0)
	}

	// Calculate total topics for progress tracking (only for selected sub-forums)
	totalTopicsToProcess := 0
	for _, sf := range sortedSubForums {
		totalTopicsToProcess += sf.TopicCount // Use TopicCount from the SubForum struct
	}
	processedTopicsSoFar := 0
	totalTopicsOverallForRun := 0 // Renamed from totalTopicsToProcess to avoid confusion with loop var
	for _, sf := range sortedSubForums {
		totalTopicsOverallForRun += sf.TopicCount
	}

	log.Println("[INFO] Starting main archival loop...")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- TEMPORARY STOPPER FOR TESTING STATE RESUMPTION ----
	// const topicsToProcessBeforeStop = 10 // Keep this line commented for future reference
	// var topicsProcessedThisSession = 0
	// ---- END TEMPORARY STOPPER ----

	skipToSubForumID := state.CurrentState.LastProcessedSubForumID
	skipToTopicID := state.CurrentState.LastProcessedTopicID
	startPageForTopic := state.CurrentState.LastProcessedPageNumberInTopic + 1
	foundResumedSubForum := false // Tracks if we have processed or skipped up to the resume subforum

	for _, subForum := range sortedSubForums {
		log.Printf("[INFO] ARCHIVER: Processing SubForum ID: %s, Name: %s, URL: %s, Topics: %d", subForum.ID, subForum.Name, subForum.URL, subForum.TopicCount)

		// Resume logic for sub-forums
		if skipToSubForumID != "" && subForum.ID != skipToSubForumID && !foundResumedSubForum {
			log.Printf("[INFO] ARCHIVER: Skipping SubForum ID %s (waiting for resume point %s).", subForum.ID, skipToSubForumID)
			processedTopicsSoFar += subForum.TopicCount // Add its topics to processed count for ETC
			continue
		}
		if subForum.ID == skipToSubForumID {
			foundResumedSubForum = true
		}
		// If we have passed the resume subforum, or if there was no resume subforum, mark as found
		if skipToSubForumID == "" {
			foundResumedSubForum = true
		}

		isSubForumCompleted := false
		for _, completedSFID := range state.CurrentState.CompletedSubForumIDs {
			if completedSFID == subForum.ID {
				isSubForumCompleted = true
				break
			}
		}
		if isSubForumCompleted {
			log.Printf("[INFO] ARCHIVER: SubForum ID %s was already marked as completed in state. Skipping.", subForum.ID)
			// Add its topics to processed count for ETC if not already counted by skip logic above
			// This can be tricky; ensure it's not double-counted.
			// If we skipped it above because it was before skipToSubForumID, it was counted.
			// If we are here because it *is* skipToSubForumID but already completed, we might need to count it.
			// Simpler: the main skip logic already counts. If we are here, it implies we didn't skip it above due to subforum ID.
			// if !(skipToSubForumID != "" && subForum.ID != skipToSubForumID && !foundResumedSubForum) {
			// 	 processedTopicsSoFar += subForum.TopicCount
			// }
			continue
		}

		if subForum.ID != state.CurrentState.LastProcessedSubForumID {
			state.CurrentState.ProcessedTopicIDsInCurrentSubForum = make([]string, 0)
		}

		// JIT Refresh Logic - Conceptual, needs actual implementation from Story 2.8 AC9
		var topicsToProcessInSubForum []data.Topic = subForum.Topics
		if cfg.JITRefreshPages > 0 && subForum.URL != "" {
			if jitrefresh.ShouldPerformJITRefresh(subForum, state.CurrentState, cfg.JITRefreshPages > 0, cfg.JITRefreshInterval) {
				log.Printf("[INFO] JITREFRESH: Performing JIT index refresh for SubForum %s (URL: %s, first %d pages)", subForum.ID, subForum.URL, cfg.JITRefreshPages)
				newlyDiscoveredTopics, errJIT := jitrefresh.PerformJITRefresh(subForum, cfg, htmlFetcher, htmlParser, htmlExtractor) // Corrected call
				if errJIT != nil {
					log.Printf("[WARNING] JITREFRESH: Error during JIT refresh for %s: %v. Proceeding with initially indexed topics.", subForum.ID, errJIT)
				} else if len(newlyDiscoveredTopics) > 0 {
					log.Printf("[INFO] JITREFRESH: Found %d new/updated topics for %s.", len(newlyDiscoveredTopics), subForum.ID)
					// Merge logic: Add new topics, potentially update existing ones. For now, just append and let processing handle duplicates if any based on ID.
					existingTopicMap := make(map[string]bool)
					for _, t := range topicsToProcessInSubForum {
						existingTopicMap[t.ID] = true
					}
					addedCount := 0
					for _, newTopic := range newlyDiscoveredTopics {
						if !existingTopicMap[newTopic.ID] {
							topicsToProcessInSubForum = append(topicsToProcessInSubForum, newTopic)
							addedCount++
						}
					}
					if addedCount > 0 {
						log.Printf("[INFO] JITREFRESH: Added %d unique new topics to process for subforum %s.", addedCount, subForum.ID)
						totalTopicsToProcess += addedCount // Adjust total for ETC
						// Re-sort if order matters after JIT additions
						sort.Slice(topicsToProcessInSubForum, func(i, j int) bool {
							return topicsToProcessInSubForum[i].ID < topicsToProcessInSubForum[j].ID
						})
					}
				}
			}
		}

		foundTopicToResumeInSF := false // Specific to this sub-forum's topic loop
		if subForum.ID != skipToSubForumID || skipToTopicID == "" {
			foundTopicToResumeInSF = true // If not the resume sub-forum, or no specific topic to resume, process all topics
		}

		for _, topic := range topicsToProcessInSubForum {
			select {
			case <-ctx.Done():
				log.Println("[INFO] ARCHIVER: Shutdown signal received. Saving state and exiting...")
				state.SaveProgress(cfg.StateFilePath)
				metrics.SaveDetailMetricsLog() // Save metrics on graceful shutdown too
				return
			default:
			}

			// ---- TEMPORARY STOPPER LOGIC ----
			/*
				if topicsProcessedThisSession >= topicsToProcessBeforeStop {
					log.Printf("[INFO] ARCHIVER: TEMPORARY STOP: Reached %d topics processed this session. Stopping to test resume.", topicsProcessedThisSession)
					state.SaveProgress(cfg.StateFilePath) // Ensure state is saved
					metrics.SaveDetailMetricsLog()
					return // Exit main
				}
			*/
			// ---- END TEMPORARY STOPPER LOGIC ----

			// Topic-level resume logic
			if !foundTopicToResumeInSF && topic.ID != skipToTopicID {
				log.Printf("[INFO] ARCHIVER: Skipping Topic ID %s in SubForum %s (waiting for resume topic %s).", topic.ID, subForum.ID, skipToTopicID)
				continue
			}
			if subForum.ID == skipToSubForumID && topic.ID == skipToTopicID {
				foundTopicToResumeInSF = true // This is the specific topic we want to start/resume from
			}

			// Check if topic was already processed in this sub-forum run (relevant for state consistency)
			isTopicProcessedInList := false
			for _, processedID := range state.CurrentState.ProcessedTopicIDsInCurrentSubForum {
				if processedID == topic.ID {
					isTopicProcessedInList = true
					break
				}
			}

			if isTopicProcessedInList {
				// If it's in the list, it means it was completed in a *previous session if we are resuming*,
				// OR it was completed *in this session* if LastProcessedSubForumID hasn't changed.
				// We need to decide if we should skip it or resume it partially.
				isTheExactResumeTopicAndSubForum := (subForum.ID == skipToSubForumID && topic.ID == skipToTopicID)

				if isTheExactResumeTopicAndSubForum && startPageForTopic > 1 {
					// This is the specific topic we are resuming, AND it was partially processed.
					// So, do NOT skip. Proceed to page-level resume.
					log.Printf("[DEBUG] ARCHIVER: Topic %s is the resume point and was partial. Proceeding to page resume from page %d.", topic.ID, startPageForTopic)
				} else {
					// It's either:
					// 1. Fully processed in a previous session (isTheExactResumeTopicAndSubForum=true, startPageForTopic=1)
					// 2. Processed in a previous session and is NOT the specific resume topic (isTheExactResumeTopicAndSubForum=false)
					// 3. Processed earlier in THIS current session (if skipToSubForumID/skipToTopicID were different or empty)
					log.Printf("[INFO] ARCHIVER: Topic ID %s (SF:%s) was already processed or is fully completed resume point. Skipping.", topic.ID, subForum.ID)
					continue
				}
			}

			currentPage := 1
			// Page-level resume logic for the specific topic we left off on (if any)
			// This condition is true only if we are on the exact subforum and topic from the state, AND that topic was partially processed.
			if subForum.ID == skipToSubForumID && topic.ID == skipToTopicID && startPageForTopic > 1 {
				log.Printf("[INFO] ARCHIVER: Resuming Topic ID: %s from page %d", topic.ID, startPageForTopic)
				currentPage = startPageForTopic
				// Reset outer resume flags once we've hit the exact resume spot for a partial topic
				// skipToSubForumID = "" // These should be reset carefully, perhaps after the topic loop or sf loop.
				// skipToTopicID = ""
				// startPageForTopic = 1 // Reset for subsequent topics
			}
			// Note: If it was the exact resume topic BUT startPageForTopic was 1 (meaning it was fully processed),
			// the 'continue' in the block above should have skipped it.

			log.Printf("[INFO] ARCHIVER: Starting processing for Topic ID: %s, Title: %s, URL: %s (SubForum: %s)", topic.ID, topic.Title, topic.URL, subForum.Name)
			pagesProcessedThisRunForTopic := 0

			// For each topic, discover all its pages
			topicPageURLs := make([]string, 0, 5) // Pre-allocate a bit
			allTopicPages := make(map[string]bool)
			pagesToScan := make([]string, 0, 5)

			// Normalize the initial topic URL from the index and add it
			normalizedInitialURL, errNormInitial := util.NormalizeTopicPageURL(topic.URL, topic.SubForumID)
			if errNormInitial != nil {
				log.Printf("[ERROR] NAV: Failed to normalize initial topic URL '%s' for topic %s: %v. Skipping this topic.", topic.URL, topic.ID, errNormInitial)
				continue // to the next topic in the sf.Topics loop
			}
			topicPageURLs = append(topicPageURLs, normalizedInitialURL)
			allTopicPages[normalizedInitialURL] = true
			pagesToScan = append(pagesToScan, normalizedInitialURL)
			log.Printf("[DEBUG] NAV: Added initial normalized page %s to scan queue for topic %s", normalizedInitialURL, topic.ID)

			// Loop to find all pages for this topic by scanning pages for pagination links
			for len(pagesToScan) > 0 {
				currentScanURL := pagesToScan[0]
				pagesToScan = pagesToScan[1:]

				log.Printf("[DEBUG] NAV: Fetching %s for pagination (topic %s)", currentScanURL, topic.ID)
				pageHTML, errFetch := htmlFetcher.FetchHTML(currentScanURL)
				if errFetch != nil {
					log.Printf("[ERROR] NAV: Error fetching %s for pagination: %v", currentScanURL, errFetch)
					continue // Skip this page if fetch fails
				}

				log.Printf("[DEBUG] NAV: Parsing %s for pagination links (topic %s)", currentScanURL, topic.ID)
				paginationLinks, errParse := htmlParser.ParsePaginationLinks(pageHTML, currentScanURL)
				if errParse != nil {
					log.Printf("[ERROR] NAV: Error parsing pagination links from %s: %v", currentScanURL, errParse)
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

			for i := currentPage - 1; i < len(topicPageURLs); i++ {
				select {
				case <-ctx.Done():
					log.Println("[INFO] ARCHIVER: Shutdown signal received. Saving state...")
					state.SaveProgress(cfg.StateFilePath)
					return
				default:
				}
				pageURL := topicPageURLs[i]
				actualPageNum := i + 1
				log.Printf("[INFO] ARCHIVER: Processing page %d/%d (URL: %s) of Topic ID: %s", actualPageNum, len(topicPageURLs), pageURL, topic.ID)
				downloadStartTime := time.Now()
				htmlBytes, err := pageDownloader.FetchPage(pageURL)
				downloadDuration := time.Since(downloadStartTime)
				if err != nil {
					log.Printf("[ERROR] ARCHIVER: Download failed for page %d (URL: %s) topic %s: %v.", actualPageNum, pageURL, topic.ID, err)
					metrics.AppendDetailMetric(metrics.PerformanceMetric{Timestamp: time.Now(), ResourceType: metrics.ResourceTypeTopicPage, ResourceID: fmt.Sprintf("%s_p%d", topic.ID, actualPageNum), Action: metrics.ActionSkipped, Size: 0, Duration: downloadDuration, Notes: fmt.Sprintf("download error: %v", err)})
					continue
				}
				filePath, err := storePageHTML(htmlStorer, topic, subForum.ID, actualPageNum, htmlBytes)
				if err != nil {
					log.Printf("[ERROR] ARCHIVER: Save failed for page %d (URL: %s) topic %s: %v.", actualPageNum, pageURL, topic.ID, err)
					metrics.AppendDetailMetric(metrics.PerformanceMetric{Timestamp: time.Now(), ResourceType: metrics.ResourceTypeTopicPage, ResourceID: fmt.Sprintf("%s_p%d", topic.ID, actualPageNum), Action: metrics.ActionSkipped, Size: 0, Duration: downloadDuration, Notes: fmt.Sprintf("save error: %v", err)})
					continue
				}
				log.Printf("[INFO] ARCHIVER: Saved HTML for topic %s, page %d to %s", topic.ID, actualPageNum, filePath)
				currentBatchMetrics.PagesArchived++
				currentBatchMetrics.BytesArchived += int64(len(htmlBytes))
				metrics.AppendDetailMetric(metrics.PerformanceMetric{Timestamp: time.Now(), ResourceType: metrics.ResourceTypeTopicPage, ResourceID: fmt.Sprintf("%s_p%d", topic.ID, actualPageNum), Action: metrics.ActionArchived, Size: int64(len(htmlBytes)), Duration: downloadDuration})
				state.CurrentState.LastProcessedPageNumberInTopic = actualPageNum
				pagesProcessedThisRunForTopic++
				if cfg.SaveStateInterval <= 0 {
					log.Printf("[DEBUG] ARCHIVER: Saving state post-page (interval <=0) for topic %s, page %d.", topic.ID, actualPageNum)
					state.SaveProgress(cfg.StateFilePath)
				}
			}
			log.Printf("[INFO] ARCHIVER: Finished Topic ID: %s. Pages processed in this run: %d", topic.ID, pagesProcessedThisRunForTopic)
			alreadyInProcessedList := false
			for _, pid := range state.CurrentState.ProcessedTopicIDsInCurrentSubForum {
				if pid == topic.ID {
					alreadyInProcessedList = true
					break
				}
			}
			if !alreadyInProcessedList {
				state.CurrentState.ProcessedTopicIDsInCurrentSubForum = append(state.CurrentState.ProcessedTopicIDsInCurrentSubForum, topic.ID)
			}
			state.CurrentState.LastProcessedTopicID = topic.ID
			state.CurrentState.LastProcessedPageNumberInTopic = 0
			state.CurrentState.LastProcessedSubForumID = subForum.ID
			currentBatchMetrics.TopicsArchived++
			// topicsProcessedThisSession++ // Increment for temporary stopper
			processedTopicsSoFar++
			log.Printf("[DEBUG] ARCHIVER: Saving state post-topic %s.", topic.ID)
			state.SaveProgress(cfg.StateFilePath)
			currentBatchMetrics.UpdateRates()
			displayProgressAndETC(processedTopicsSoFar, totalTopicsOverallForRun, currentBatchMetrics)
			if time.Since(lastMetricsSave) > 5*time.Minute {
				log.Println("[INFO] ARCHIVER: Periodically saving detailed performance metrics...")
				if err := metrics.SaveDetailMetricsLog(); err != nil {
					log.Printf("[ERROR] ARCHIVER: Periodic metrics save failed: %v", err)
				}
				lastMetricsSave = time.Now()
			}
		} // End topic loop

		log.Printf("[INFO] ARCHIVER: Finished SubForum ID: %s, Name: %s", subForum.ID, subForum.Name)
		alreadyCompletedSF := false
		for _, csfID := range state.CurrentState.CompletedSubForumIDs {
			if csfID == subForum.ID {
				alreadyCompletedSF = true
				break
			}
		}
		if !alreadyCompletedSF {
			state.CurrentState.CompletedSubForumIDs = append(state.CurrentState.CompletedSubForumIDs, subForum.ID)
		}
		state.CurrentState.ProcessedTopicIDsInCurrentSubForum = make([]string, 0)
		log.Printf("[DEBUG] ARCHIVER: Saving state post-subforum %s.", subForum.ID)
		state.SaveProgress(cfg.StateFilePath)
	} // End sub-forum loop

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
