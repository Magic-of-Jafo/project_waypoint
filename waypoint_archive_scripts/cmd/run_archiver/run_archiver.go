package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
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
)

var (
	cfg *config.Config // Global config instance
)

// getAllPageURLsForTopic fetches the first page of a topic, parses pagination links,
// and returns a list of all page URLs for that topic.
// It now uses htmlutil.FetchHTMLer and htmlutil.ParsePaginationLinker interfaces.
func getAllPageURLsForTopic(topicURL string, fetcher htmlutil.FetchHTMLer, parser htmlutil.ParsePaginationLinker) ([]string, error) {
	if topicURL == "" {
		return nil, fmt.Errorf("getAllPageURLsForTopic: topicURL cannot be empty")
	}
	if fetcher == nil {
		return nil, fmt.Errorf("getAllPageURLsForTopic: fetcher cannot be nil")
	}
	if parser == nil {
		return nil, fmt.Errorf("getAllPageURLsForTopic: parser cannot be nil")
	}

	log.Printf("[DEBUG] getAllPageURLsForTopic: Fetching initial page for URL: %s", topicURL)
	htmlContent, err := fetcher.FetchHTML(topicURL)
	if err != nil {
		log.Printf("[ERROR] getAllPageURLsForTopic: Failed to fetch HTML for %s: %v", topicURL, err)
		return nil, fmt.Errorf("failed to fetch initial topic page %s: %w", topicURL, err)
	}

	pageURLs, err := parser.ParsePaginationLinks(htmlContent, topicURL)
	if err != nil {
		log.Printf("[ERROR] getAllPageURLsForTopic: Failed to parse pagination links for %s: %v", topicURL, err)
		// If parsing fails but we have the first page, we can at least archive that.
		// Consider if this should be a fatal error for the topic or just return the base URL.
		// For now, if parsing fails, we assume it's a single-page topic or the structure is unexpected.
		// Return just the base URL.
		return []string{topicURL}, nil // Or return the error: fmt.Errorf("failed to parse pagination for %s: %w", topicURL, err)
	}

	if len(pageURLs) == 0 { // Should always include the base URL itself
		log.Printf("[WARN] getAllPageURLsForTopic: No page URLs (not even base) returned from parser for %s. Defaulting to topicURL.", topicURL)
		return []string{topicURL}, nil
	}

	log.Printf("[DEBUG] getAllPageURLsForTopic: Found %d page(s) for topic %s", len(pageURLs), topicURL)
	return pageURLs, nil
}

func displayProgressAndETC(batchMetrics *metrics.BatchMetrics, totalTopicsToProcess, totalPagesToProcess int, processingStartTime time.Time) {
	processedTopics := batchMetrics.TopicsArchived      // Direct field access
	processedPages := batchMetrics.PagesArchived        // Direct field access
	errorsEncountered := batchMetrics.ErrorsEncountered // Direct field access

	topicsRemaining := totalTopicsToProcess - int(processedTopics) // Cast to int for calculation
	pagesRemaining := totalPagesToProcess - int(processedPages)    // Cast to int for calculation

	elapsedTime := time.Since(processingStartTime)
	var averageTimePerTopic, averageTimePerPage time.Duration
	var estimatedTimeRemaining time.Duration

	if processedTopics > 0 {
		averageTimePerTopic = elapsedTime / time.Duration(processedTopics)
	}
	if processedPages > 0 {
		averageTimePerPage = elapsedTime / time.Duration(processedPages)
	}

	if processedTopics > 0 && topicsRemaining > 0 {
		estimatedTimeRemaining = averageTimePerTopic * time.Duration(topicsRemaining)
	} else if processedPages > 0 && pagesRemaining > 0 { // Fallback if topic-level ETC is not available
		estimatedTimeRemaining = averageTimePerPage * time.Duration(pagesRemaining)
	}

	// Prepare output string
	progressStr := fmt.Sprintf(
		"[PROGRESS] Elapsed: %s | Topics: %d/%d (%d rem) | Pages: %d/%d (%d rem) | Avg/Topic: %s | Avg/Page: %s",
		elapsedTime.Round(time.Second),
		processedTopics, totalTopicsToProcess, topicsRemaining,
		processedPages, totalPagesToProcess, pagesRemaining,
		averageTimePerTopic.Round(time.Millisecond),
		averageTimePerPage.Round(time.Millisecond),
	)
	if estimatedTimeRemaining > 0 {
		progressStr += fmt.Sprintf(" | ETC: %s", estimatedTimeRemaining.Round(time.Second))
	}
	if errorsEncountered > 0 {
		progressStr += fmt.Sprintf(" | Errors: %d", errorsEncountered)
	}

	log.Println(progressStr)
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.Println("[INFO] Waypoint Archiver starting...")

	var err error
	cfg, err = config.LoadConfig(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// flag.ErrHelp is special, means -help was called.
			// config.LoadConfig should have printed usage. We just exit.
			os.Exit(0)
		}
		log.Fatalf("[FATAL] Error loading configuration: %v", err)
	}

	// Setup logging level and output file (if specified)
	// This would ideally be handled by a dedicated logging package, but for now:
	if cfg.LogFilePath != "" {
		logFile, err := os.OpenFile(cfg.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("[FATAL] Failed to open log file %s: %v", cfg.LogFilePath, err)
		}
		defer logFile.Close()
		log.SetOutput(logFile)
		log.Printf("[INFO] Logging to file: %s", cfg.LogFilePath)
	}
	// Setting log level is more complex with standard logger, would need a wrapper or custom solution.
	// For now, using verbose logs and relying on INFO/ERROR/DEBUG prefixes.

	log.Printf("[INFO] Configuration loaded. UserAgent: %s, PolitenessDelay: %s", cfg.UserAgent, cfg.PolitenessDelay)

	// Initialize Metrics Logger (uses cfg.PerformanceLogPath)
	metrics.InitPerformanceLogger(cfg.PerformanceLogPath)
	// Ensure metrics are saved on exit, especially if panic or early exit occurs.
	// Note: Signal handler already tries to save. This is an additional guard.
	// defer metrics.SaveDetailMetricsLog() // This might be redundant with signal handler and explicit final save.

	if len(cfg.TestSubForumIDs) > 0 {
		log.Printf("[INFO] TEST MODE ACTIVE. Processing only SubForumIDs: %v", cfg.TestSubForumIDs)
		log.Printf("[INFO] TEST MODE: Archive output will be rooted at: %s", cfg.TestArchiveOutputRoot)
		cfg.ArchiveOutputRootDir = cfg.TestArchiveOutputRoot // Override main archive root for test mode
	}

	// Initialize components
	dl := downloader.NewDownloader(cfg)
	htmlStore := storer.NewStorer(cfg) // Uses cfg.ArchiveOutputRootDir

	// --- Load SubForum List and Topic Indices ---
	log.Printf("[INFO] Loading sub-forum list from: %s", cfg.SubForumListFile)
	subForumsMap, err := indexerlogic.ReadSubForumListCSV(cfg.SubForumListFile)
	if err != nil {
		log.Fatalf("[FATAL] Failed to read sub-forum list from %s: %v", cfg.SubForumListFile, err)
	}
	log.Printf("[INFO] Successfully loaded %d unique sub-forums from CSV.", len(subForumsMap))

	var allSubForumsList []data.SubForum
	var allTopicsMasterList []data.Topic

	targetSubForumIDs := make(map[string]bool)
	if len(cfg.TestSubForumIDs) > 0 {
		for _, id := range cfg.TestSubForumIDs {
			targetSubForumIDs[id] = true
		}
	} else {
		for sfID := range subForumsMap {
			targetSubForumIDs[sfID] = true
		}
	}
	log.Printf("[INFO] Will attempt to process %d target sub-forums.", len(targetSubForumIDs))

	for sfID, sfDataFromCSV := range subForumsMap {
		if !targetSubForumIDs[sfID] {
			log.Printf("[DEBUG] Skipping SubForumID %s as it's not in the target list.", sfID)
			continue
		}

		var topicsForSubForum []data.Topic
		var errReadTopics error

		var topicIndexFilename string
		// Check if TopicIndexFilePattern contains a path separator, implying a nested structure like "forum_%s/topic_index_%s.json"
		if strings.Contains(cfg.TopicIndexFilePattern, string(os.PathSeparator)) || strings.Contains(cfg.TopicIndexFilePattern, "/") {
			// Assumes pattern like "forum_%s/topic_index_%s.json" where both %s are sfID
			topicIndexFilename = fmt.Sprintf(cfg.TopicIndexFilePattern, sfID, sfID)
		} else {
			// Assumes pattern like "topic_index_forum_%s.csv" or "topic_index_%s.json"
			topicIndexFilename = fmt.Sprintf(cfg.TopicIndexFilePattern, sfID)
		}
		topicIndexFilePath := filepath.Join(cfg.TopicIndexDir, topicIndexFilename)

		log.Printf("[INFO] Attempting to load topic index for SubForumID %s using path: %s", sfID, topicIndexFilePath)

		if strings.HasSuffix(strings.ToLower(topicIndexFilePath), ".json") {
			log.Printf("[DEBUG] Identified JSON topic index file for SubForumID %s: %s", sfID, topicIndexFilePath)
			topicsForSubForum, errReadTopics = indexerlogic.ReadTopicIndexJSON(topicIndexFilePath, sfID)
		} else if strings.HasSuffix(strings.ToLower(topicIndexFilePath), ".csv") {
			log.Printf("[DEBUG] Identified CSV topic index file for SubForumID %s: %s", sfID, topicIndexFilePath)
			topicsForSubForum, errReadTopics = indexerlogic.ReadTopicIndexCSV(topicIndexFilePath, sfID)
		} else {
			log.Printf("[WARN] Unrecognized topic index file extension for %s. Pattern was '%s'. Skipping.", topicIndexFilePath, cfg.TopicIndexFilePattern)
			errReadTopics = fmt.Errorf("unrecognized file type for topic index: %s (pattern: %s)", topicIndexFilePath, cfg.TopicIndexFilePattern)
		}

		if errReadTopics != nil {
			log.Printf("[ERROR] Failed to read topic index for SubForumID %s from %s: %v. Skipping this sub-forum's topics.", sfID, topicIndexFilePath, errReadTopics)
			// We still create a SubForum entry, but it will have no topics.
			topicsForSubForum = []data.Topic{} // Ensure it's an empty slice
		}

		if len(topicsForSubForum) == 0 && errReadTopics == nil { // errReadTopics == nil means file was found and read, but was empty or had no valid topics
			log.Printf("[INFO] No topics found or loaded for SubForumID %s from %s.", sfID, topicIndexFilePath)
		} else if errReadTopics == nil { // Only log success if there wasn't an error during read that we're skipping past
			log.Printf("[INFO] Successfully loaded %d topics for SubForumID %s from %s.", len(topicsForSubForum), sfID, topicIndexFilePath)
		}

		currentSubForum := data.SubForum{
			ID:         sfID,
			Name:       sfDataFromCSV.Name,
			URL:        sfDataFromCSV.URL,
			Topics:     topicsForSubForum,
			TopicCount: len(topicsForSubForum),
		}
		allSubForumsList = append(allSubForumsList, currentSubForum)
		allTopicsMasterList = append(allTopicsMasterList, topicsForSubForum...)
	}
	log.Printf("[INFO] Finished loading all topic indices. Total topics loaded across all sub-forums: %d", len(allTopicsMasterList))
	if len(allTopicsMasterList) == 0 {
		log.Fatalf("[FATAL] No topics loaded from any sub-forum. Check configuration for TopicIndexDir ('%s') and TopicIndexFilePattern ('%s'). Exiting.", cfg.TopicIndexDir, cfg.TopicIndexFilePattern)
	}
	// --- End Load SubForum List and Topic Indices ---

	archivalState, err := state.LoadState(cfg.StateFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("[INFO] No existing state file found at %s. Creating new state.", cfg.StateFilePath)
			archivalState = state.NewArchiveProgressState()
		} else {
			log.Fatalf("[FATAL] Failed to load archival state from %s: %v", cfg.StateFilePath, err)
		}
	} else {
		log.Printf("[INFO] Successfully loaded archival state from %s. %d topics and %d pages previously archived.",
			cfg.StateFilePath, len(archivalState.ArchivedTopics), archivalState.TotalPagesArchived())
	}

	// Initialize Metrics
	batchMetrics := metrics.NewBatchMetrics()
	// Assuming PerformanceLogPath is for detailed, line-by-line metrics.
	// If it's for batch summary, adjust accordingly.
	// detailMetricsLog, err := metrics.NewDetailMetricsLog(cfg.PerformanceLogPath) // Removed
	// if err != nil { // Removed
	// 	log.Fatalf("[FATAL] Failed to initialize detail metrics log at %s: %v", cfg.PerformanceLogPath, err) // Removed
	// } // Removed
	// defer detailMetricsLog.Close() // Removed, replaced by SaveDetailMetricsLog in appropriate places

	// Graceful shutdown setup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("[INFO] Received signal: %v. Shutting down gracefully...", sig)
		cancel() // Trigger context cancellation
		// Perform any other cleanup needed before forced exit if wg.Wait timeouts
		// For example, try to save state one last time.
		log.Printf("[INFO] Attempting final state save on shutdown...")
		if err := archivalState.Save(cfg.StateFilePath); err != nil {
			log.Printf("[ERROR] Failed to save state during shutdown: %v", err)
		} else {
			log.Printf("[INFO] State saved successfully during shutdown.")
		}
		log.Printf("[INFO] Attempting final detail metrics log save on shutdown...")
		// if err := detailMetricsLog.Save(); err != nil { // Assuming Save exists if needed - old way
		if err := metrics.SaveDetailMetricsLog(); err != nil { // New way
			log.Printf("[ERROR] Failed to save detail metrics log during shutdown: %v", err)
		} else {
			log.Printf("[INFO] Detail metrics log saved successfully during shutdown.")
		}

	}()

	// --- Main Archival Loop ---
	log.Printf("[INFO] Starting main archival loop for %d topics...", len(allTopicsMasterList))
	processingStartTime := time.Now()
	lastStateSaveTime := time.Now()
	var totalPagesEstimated int = 0 // Basic estimation, will be refined if possible

	// Pre-calculate totalPagesEstimated if possible (very rough)
	for _, topic := range allTopicsMasterList {
		// Rough estimate: average 1 page for new topics, or actual pages if known
		// This is not accurate without fetching, so it's a placeholder.
		// We could improve this if Topic struct had a PageCount field from indexer.
		if topic.Replies > 0 && cfg.ForumBaseURL != "" { // Assuming some posts per page
			// A very rough estimate, e.g., 20 posts per page
			totalPagesEstimated += (topic.Replies / 20) + 1
		} else {
			totalPagesEstimated++
		}
	}

	for i, topic := range allTopicsMasterList {
		var err error // Declare err once for the topic processing scope

		select {
		case <-ctx.Done():
			log.Printf("[INFO] Context cancelled. Exiting archival loop.")
			goto endLoop // Use goto to break out of nested loops and proceed to cleanup
		default:
			// Continue processing
		}

		log.Printf("[INFO] Processing topic %d/%d: ID %s, Title: %s", i+1, len(allTopicsMasterList), topic.ID, topic.Title)

		if archivalState.IsTopicArchived(topic.ID) {
			log.Printf("[INFO] Topic %s (ID: %s) already archived. Skipping.", topic.Title, topic.ID)
			batchMetrics.TopicsSkipped++ // Direct field increment
			// Ensure it counts towards "processed" for ETC calculation stability
			// batchMetrics.TopicsArchived++ // Or have a separate "processed" counter for ETC
			continue
		}

		// Construct topic URL if not absolute or missing scheme
		var currentTopicURL string
		if strings.HasPrefix(topic.URL, "http://") || strings.HasPrefix(topic.URL, "https://") {
			currentTopicURL = topic.URL
		} else if cfg.ForumBaseURL != "" {
			parsedBaseURL, err := url.Parse(cfg.ForumBaseURL)
			if err != nil {
				log.Printf("[ERROR] Invalid ForumBaseURL '%s': %v. Cannot construct full URL for topic %s (%s). Skipping topic.", cfg.ForumBaseURL, err, topic.Title, topic.ID)
				metrics.AppendDetailMetric(metrics.PerformanceMetric{
					Timestamp:    time.Now(),
					ResourceType: metrics.ResourceTypeTopicPage,
					ResourceID:   topic.ID,
					Action:       metrics.ActionProcessTopic,
					Duration:     time.Since(processingStartTime),
					Notes:        fmt.Sprintf("Status: Error, Invalid ForumBaseURL: %v", err),
				})
				continue
			}
			// Ensure topic.URL is treated as a path relative to the base
			// url.ResolveReference expects the reference to be a valid URI reference.
			// If topic.URL is like "viewtopic.php?t=123", it should be fine.
			// If it's just "123", it might not resolve as expected.
			// Assuming topic.URL is a relative path from the forum root.
			parsedTopicPath, err := url.Parse(topic.URL)
			if err != nil {
				log.Printf("[ERROR] Invalid Topic URL fragment '%s': %v. Cannot construct full URL for topic %s (%s). Skipping topic.", topic.URL, err, topic.Title, topic.ID)
				metrics.AppendDetailMetric(metrics.PerformanceMetric{
					Timestamp:    time.Now(),
					ResourceType: metrics.ResourceTypeTopicPage,
					ResourceID:   topic.ID,
					Action:       metrics.ActionProcessTopic,
					Duration:     time.Since(processingStartTime),
					Notes:        fmt.Sprintf("Status: Error, Invalid topic URL fragment: %v", err),
				})
				continue
			}
			currentTopicURL = parsedBaseURL.ResolveReference(parsedTopicPath).String()
			log.Printf("[DEBUG] Constructed topic URL: %s (from base: %s and topic part: %s)", currentTopicURL, cfg.ForumBaseURL, topic.URL)
		} else {
			log.Printf("[ERROR] Topic URL for %s (%s) is relative ('%s') but ForumBaseURL is not configured. Skipping topic.", topic.Title, topic.ID, topic.URL)
			metrics.AppendDetailMetric(metrics.PerformanceMetric{
				Timestamp:    time.Now(),
				ResourceType: metrics.ResourceTypeTopicPage,
				ResourceID:   topic.ID,
				Action:       metrics.ActionProcessTopic,
				Duration:     time.Since(processingStartTime),
				Notes:        "Status: Error, Relative topic URL with no ForumBaseURL",
			})
			continue
		}

		// JIT Refresh Logic (Placeholder - needs actual SubForum context for the topic)
		// This needs to find the SubForum object to which this topic belongs to use subForum.JITRefreshCandidate
		// For now, this is simplified. A more robust solution would involve mapping topics back to their subforums
		// or passing subforum context through.
		// We'll assume that if JITRefreshPages > 0, we *might* do it.
		if cfg.JITRefreshPages > 0 { // Basic check, refined by ShouldPerformJITRefresh
			// Find the subforum for the current topic
			var parentSubForum data.SubForum
			foundSF := false
			for _, sf := range allSubForumsList {
				if sf.ID == topic.SubForumID {
					parentSubForum = sf
					foundSF = true
					break
				}
			}

			if foundSF && jitrefresh.ShouldPerformJITRefresh(parentSubForum, archivalState, cfg.JITRefreshPages > 0, cfg.JITRefreshInterval) {
				log.Printf("[INFO] Performing JIT Refresh for SubForum %s (topic %s is part of it)", parentSubForum.ID, topic.ID)
				// Prepare interfaces for JIT refresh using new constructors from htmlutil
				htmlFetcherForJIT := htmlutil.NewHTMLFetcher(cfg.UserAgent, cfg.PolitenessDelay)
				paginationParserForJIT := htmlutil.NewPaginationParser(cfg.ForumBaseURL)
				topicExtractorForJIT := htmlutil.NewTopicExtractor(cfg.ForumBaseURL) // Provides htmlutil.ExtractTopicser

				refreshedTopics, err := jitrefresh.PerformJITRefresh(
					parentSubForum,
					cfg, // Pass the whole config object
					htmlFetcherForJIT,
					paginationParserForJIT,
					topicExtractorForJIT,
				)
				if err != nil {
					log.Printf("[ERROR] JIT Refresh for SubForum %s failed: %v", parentSubForum.ID, err)
					metrics.AppendDetailMetric(metrics.PerformanceMetric{
						Timestamp:    time.Now(),
						ResourceType: metrics.ResourceTypeSubForum,
						ResourceID:   parentSubForum.ID,
						Action:       metrics.ActionJITRefresh,
						Notes:        fmt.Sprintf("Status: Error, JIT Refresh failed: %v", err),
					})
				} else {
					log.Printf("[INFO] JIT Refresh for SubForum %s completed. Found %d new/updated topics.", parentSubForum.ID, len(refreshedTopics))
					// TODO: Integrate refreshedTopics back into allTopicsMasterList or handle them
					// This might involve updating existing topic entries or adding new ones.
					// For now, we just log. A proper merge/update is complex.
					// Mark that JIT refresh was attempted/done for this sub-forum in state
					archivalState.MarkJITRefreshAttempted(parentSubForum.ID, time.Now())
					// Log successful JIT refresh completion
					metrics.AppendDetailMetric(metrics.PerformanceMetric{
						Timestamp:    time.Now(),
						ResourceType: metrics.ResourceTypeSubForum,
						ResourceID:   parentSubForum.ID,
						Action:       metrics.ActionJITRefresh,
						Notes:        fmt.Sprintf("Status: Success, Found %d new/updated topics", len(refreshedTopics)),
					})
				}
			}
		}
		// End JIT Refresh Logic

		// Use the new htmlutil constructors for fetching and parsing pagination for this specific task
		topicSpecificFetcher := htmlutil.NewHTMLFetcher(cfg.UserAgent, cfg.PolitenessDelay)
		topicSpecificPaginationParser := htmlutil.NewPaginationParser(cfg.ForumBaseURL)
		topicPageURLs, err := getAllPageURLsForTopic(currentTopicURL, topicSpecificFetcher, topicSpecificPaginationParser)
		if err != nil {
			log.Printf("[ERROR] Failed to get all page URLs for topic %s (ID: %s, URL: %s): %v. Skipping topic.", topic.Title, topic.ID, currentTopicURL, err)
			// metrics.RecordPerformance(detailMetricsLog, "GetPageURLs", "Error", time.Since(processingStartTime), topic.ID, fmt.Sprintf("Failed to get page URLs: %v", err)) - Old way
			metrics.AppendDetailMetric(metrics.PerformanceMetric{
				Timestamp:    time.Now(),
				ResourceType: metrics.ResourceTypeTopicPage,
				ResourceID:   topic.ID,
				Action:       metrics.ActionGetPageURLs,
				Duration:     time.Since(processingStartTime), // processingStartTime might not be right here, maybe time.Since(topicProcessingStartTime)
				Notes:        fmt.Sprintf("Status: Error, Failed to get page URLs: %v", err),
			})
			batchMetrics.ErrorsEncountered++ // Direct field increment
			continue
		}

		log.Printf("[INFO] Topic %s (ID: %s) has %d page(s) to archive.", topic.Title, topic.ID, len(topicPageURLs))

		for pageIdx, pageURL := range topicPageURLs {
			select {
			case <-ctx.Done():
				log.Printf("[INFO] Context cancelled during page processing for topic %s. Exiting.", topic.ID)
				goto endLoop
			default:
				// Continue
			}

			pageNum := pageIdx + 1 // 1-indexed page number
			pageFetchStartTime := time.Now()

			if archivalState.IsPageArchived(topic.ID, pageNum) {
				log.Printf("[DEBUG] Page %d of topic %s (ID: %s) already archived. Skipping.", pageNum, topic.Title, topic.ID)
				// batchMetrics.PagesSkipped++ // No, this should count towards total pages for ETC.
				continue
			}

			log.Printf("[DEBUG] Fetching page %d/%d for topic %s (ID: %s) from %s", pageNum, len(topicPageURLs), topic.Title, topic.ID, pageURL)
			var htmlContent []byte
			htmlContent, err = dl.FetchPage(pageURL)
			fetchDuration := time.Since(pageFetchStartTime)
			if err != nil {
				log.Printf("[ERROR] Failed to fetch page %d of topic %s (URL: %s): %v", pageNum, topic.Title, pageURL, err)
				// metrics.RecordPerformance(detailMetricsLog, "FetchPage", "Error", fetchDuration, topic.ID, fmt.Sprintf("URL: %s, Page: %d, Error: %v", pageURL, pageNum, err)) - Old way
				metrics.AppendDetailMetric(metrics.PerformanceMetric{
					Timestamp:    time.Now(),
					ResourceType: metrics.ResourceTypeTopicPage,
					ResourceID:   topic.ID,
					Action:       metrics.ActionFetchPage,
					Duration:     fetchDuration,
					Notes:        fmt.Sprintf("Status: Error, URL: %s, Page: %d, OriginalError: %v", pageURL, pageNum, err),
				})
				batchMetrics.ErrorsEncountered++ // Direct field increment
				// Potentially skip to next page or next topic depending on error severity
				// For now, we skip this page and continue with others for the topic.
				continue
			}
			// metrics.RecordPerformance(detailMetricsLog, "FetchPage", "Success", fetchDuration, topic.ID, fmt.Sprintf("URL: %s, Page: %d, Bytes: %d", pageURL, pageNum, len(htmlContent))) - Old way
			metrics.AppendDetailMetric(metrics.PerformanceMetric{
				Timestamp:    time.Now(),
				ResourceType: metrics.ResourceTypeTopicPage,
				ResourceID:   topic.ID,
				Action:       metrics.ActionFetchPage,
				Size:         int64(len(htmlContent)),
				Duration:     fetchDuration,
				Notes:        fmt.Sprintf("Status: Success, URL: %s, Page: %d", pageURL, pageNum),
			})

			// Store the HTML content
			storageStartTime := time.Now()
			sfIDForStorage := topic.SubForumID
			if sfIDForStorage == "" {
				log.Printf("[WARN] Topic ID %s has no SubForumID associated. Using 'unknown_subforum' for storage path.", topic.ID)
				sfIDForStorage = "unknown_subforum"
			}

			_, err = htmlStore.SaveTopicHTML(sfIDForStorage, topic.ID, pageNum, htmlContent)
			storeDuration := time.Since(storageStartTime)
			if err != nil {
				log.Printf("[ERROR] Failed to store HTML for page %d of topic %s (ID: %s): %v", pageNum, topic.Title, topic.ID, err)
				// metrics.RecordPerformance(detailMetricsLog, "SaveTopicHTML", "Error", storeDuration, topic.ID, fmt.Sprintf("Page: %d, Error: %v", pageNum, err)) - Old way
				metrics.AppendDetailMetric(metrics.PerformanceMetric{
					Timestamp:    time.Now(),
					ResourceType: metrics.ResourceTypeTopicPage,
					ResourceID:   topic.ID,
					Action:       metrics.ActionSaveTopicHTML,
					Duration:     storeDuration,
					Notes:        fmt.Sprintf("Status: Error, Page: %d, OriginalError: %v", pageNum, err),
				})
				batchMetrics.ErrorsEncountered++ // Direct field increment
				continue                         // Skip this page if storage fails
			}
			// metrics.RecordPerformance(detailMetricsLog, "SaveTopicHTML", "Success", storeDuration, topic.ID, fmt.Sprintf("Page: %d", pageNum)) - Old way
			metrics.AppendDetailMetric(metrics.PerformanceMetric{
				Timestamp:    time.Now(),
				ResourceType: metrics.ResourceTypeTopicPage,
				ResourceID:   topic.ID,
				Action:       metrics.ActionSaveTopicHTML,
				Duration:     storeDuration,
				Notes:        fmt.Sprintf("Status: Success, Page: %d", pageNum),
			})
			log.Printf("[INFO] Successfully fetched and stored page %d of topic %s (ID: %s).", pageNum, topic.Title, topic.ID)

			archivalState.MarkPageAsArchived(topic.ID, pageNum, pageURL)
			batchMetrics.PagesArchived++ // Direct field increment

			// Optional: Politeness delay already handled by downloader, but an additional one here if needed.
			// time.Sleep(cfg.PolitenessDelay) // This might be too much if downloader already does it.
		}

		archivalState.MarkTopicAsArchived(topic.ID)
		batchMetrics.TopicsArchived++ // Direct field increment
		log.Printf("[INFO] Finished archiving all pages for topic %s (ID: %s).", topic.Title, topic.ID)

		// Save state periodically
		if time.Since(lastStateSaveTime) >= cfg.SaveStateInterval {
			log.Printf("[INFO] Save state interval reached. Saving progress...")
			if err := archivalState.Save(cfg.StateFilePath); err != nil {
				log.Printf("[ERROR] Failed to save state: %v", err)
				// Continue, non-fatal for the archival process itself
			} else {
				log.Printf("[INFO] State saved successfully to %s.", cfg.StateFilePath)
				lastStateSaveTime = time.Now()
			}
			// if err := detailMetricsLog.Save(); err != nil { // Persist metrics log too - old way
			if err := metrics.SaveDetailMetricsLog(); err != nil { // New way
				log.Printf("[ERROR] Failed to save detail metrics log: %v", err)
			} else {
				log.Printf("[INFO] Detail metrics log flushed to %s", cfg.PerformanceLogPath)
			}
		}

		// Display progress
		// Use totalTopicsMasterList and a rough estimate for total pages if available
		displayProgressAndETC(batchMetrics, len(allTopicsMasterList), totalPagesEstimated, processingStartTime)
	}

endLoop:
	log.Println("[INFO] Archival loop finished or was interrupted.")

	// Final save of state and metrics
	log.Printf("[INFO] Performing final state save...")
	if err := archivalState.Save(cfg.StateFilePath); err != nil {
		log.Printf("[ERROR] Final state save failed: %v", err)
	} else {
		log.Printf("[INFO] Final state saved successfully to %s.", cfg.StateFilePath)
	}

	log.Printf("[INFO] Performing final detail metrics log save...")
	// if err := detailMetricsLog.Save(); err != nil { // Ensure metrics are flushed - old way
	if err := metrics.SaveDetailMetricsLog(); err != nil { // New way
		log.Printf("[ERROR] Final save of detail metrics log failed: %v", err)
	} else {
		log.Printf("[INFO] Detail metrics log saved successfully to %s", cfg.PerformanceLogPath)
	}

	// Display final batch metrics
	log.Println("----------------------------------------------------")
	log.Println("[INFO] Archival Process Summary:")
	log.Println("----------------------------------------------------")
	log.Printf("[INFO] Total processing time: %s", time.Since(processingStartTime).Round(time.Second))
	log.Printf("[INFO] Total topics processed/attempted: %d", batchMetrics.TopicsArchived+batchMetrics.TopicsSkipped)
	log.Printf("[INFO] Successfully archived topics: %d", batchMetrics.TopicsArchived)
	log.Printf("[INFO] Skipped topics (already archived): %d", batchMetrics.TopicsSkipped)
	log.Printf("[INFO] Total pages archived: %d", batchMetrics.PagesArchived)
	log.Printf("[INFO] Errors encountered: %d", batchMetrics.ErrorsEncountered)
	log.Println("----------------------------------------------------")

	if ctx.Err() != nil {
		log.Printf("[INFO] Archiver shutdown due to context cancellation (likely signal).")
		os.Exit(1) // Indicate non-normal exit
	}

	log.Println("[INFO] Waypoint Archiver finished successfully.")
}
