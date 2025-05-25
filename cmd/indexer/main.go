package main

import (
	"flag" // Added for configuration management
	"fmt"
	"io" // New: For io.MultiWriter
	"log"
	"os"            // New: For ordered output
	"path/filepath" // New: For joining paths
	"time"          // Added for politeness delay

	// "sort" // Will be needed later for ordered output if desired

	"project-waypoint/internal/indexer/logger"  // Added custom logger
	"project-waypoint/internal/indexer/metrics" // Added for performance tracking
	"project-waypoint/internal/indexer/navigation"
	"project-waypoint/internal/indexer/storage" // Added for saving topic index
	"project-waypoint/internal/indexer/topic"
)

// Configuration struct to hold all configurable parameters
type Config struct {
	subForumURL  string
	outputDir    string
	requestDelay int // in milliseconds
	logLevel     string
	maxPages     int // New: Max pages to process for testing; 0 for no limit
}

// loadConfig loads configuration from command-line flags
func loadConfig() Config {
	cfg := Config{}

	flag.StringVar(&cfg.subForumURL, "url", "", "Target sub-forum base URL (required)")
	flag.StringVar(&cfg.outputDir, "output", "./output_data", "Base output directory for generated files")
	flag.IntVar(&cfg.requestDelay, "delay", 1000, "Delay between HTTP requests in milliseconds")
	flag.StringVar(&cfg.logLevel, "loglevel", "INFO", "Logging verbosity (DEBUG, INFO, WARNING, ERROR)")
	flag.IntVar(&cfg.maxPages, "maxpages", 0, "Maximum number of pages to process (0 for no limit, for testing)") // New flag

	flag.Parse()

	if cfg.subForumURL == "" {
		// Use standard log here as our logger might not be initialized yet, or for early critical errors.
		fmt.Fprintf(os.Stderr, "Error: Target sub-forum URL (-url) is required.\n")
		flag.Usage()
		os.Exit(1)
	}

	// Logger will print its own init message including the level.
	// logger.Infof("Configuration loaded: URL=%s, OutputDir=%s, Delay=%dms, LogLevel=%s",
	// 	cfg.subForumURL, cfg.outputDir, cfg.requestDelay, cfg.logLevel)
	return cfg
}

// performFullScan fetches all pages for a given sub-forum URL, extracts topics from each page,
// and returns a map of unique topics found, and the list of page URLs discovered.
// This function integrates parts of Story 1.1 (navigation) and Story 1.2 (topic extraction for the full scan)
func performFullScan(baseURL string, requestDelayMs int, tracker *metrics.MetricsTracker, cfgMaxPages int) (map[string]topic.TopicInfo, []string, error) {
	logger.Infof("Full Scan: Fetching initial page to discover all page URLs: %s", baseURL)

	tracker.IncrementHTTPRequests() // Track request for initial page
	initialHTMLContent, err := navigation.FetchHTML(baseURL)
	if err != nil {
		tracker.IncrementFailedRequests()
		return nil, nil, fmt.Errorf("full scan: failed to fetch HTML from %s: %w", baseURL, err)
	}
	tracker.IncrementSuccessfulRequests()

	// Apply delay after the first fetch if there will be more fetches in this function or soon after.
	// Since ParsePaginationLinks is next and then a loop of fetches, good to have a delay here.
	logger.Debugf("Full Scan: Applying %dms delay after fetching initial page...", requestDelayMs)
	time.Sleep(time.Duration(requestDelayMs) * time.Millisecond)

	logger.Infof("Full Scan: Parsing pagination links...")
	pageURLs, err := navigation.ParsePaginationLinks(initialHTMLContent, baseURL)
	if err != nil {
		// This is a parsing error, not an HTTP error for this specific step
		return nil, nil, fmt.Errorf("full scan: failed to parse pagination links from %s: %w", baseURL, err)
	}
	logger.Infof("Full Scan: Discovered %d page URLs for sub-forum.", len(pageURLs))

	// Apply maxPages limit if set
	if cfgMaxPages > 0 && len(pageURLs) > cfgMaxPages {
		logger.Warnf("Max pages limit active: Truncating page list from %d to %d pages.", len(pageURLs), cfgMaxPages)
		pageURLs = pageURLs[:cfgMaxPages]
	}
	tracker.SetTotalPages(len(pageURLs)) // Set total pages for ETC (original or truncated)

	scannedTopics := make(map[string]topic.TopicInfo)
	pageCounter := 0

	for _, pageURL := range pageURLs {
		pageCounter++
		logger.Infof("Full Scan: Processing page %d/%d: %s", pageCounter, len(pageURLs), pageURL)

		tracker.IncrementHTTPRequests() // Track request for each page in loop
		htmlContent, err := navigation.FetchHTML(pageURL)
		if err != nil {
			tracker.IncrementFailedRequests()
			logger.Warnf("Full Scan: Error fetching HTML from %s: %v. Skipping this page.", pageURL, err)
			// Optional: Could add a specific shorter error-delay here if needed
			continue // Or implement retry logic based on Story/Operational Guidelines
		}
		tracker.IncrementSuccessfulRequests()
		tracker.IncrementPagesFetched() // Page successfully fetched and will be processed

		topicsOnPage, err := topic.ExtractTopics(htmlContent, pageURL) // Story 1.2 integration
		if err != nil {
			// This is a parsing error for this page
			logger.Warnf("Full Scan: Error extracting topics from %s: %v. Skipping this page.", pageURL, err)
			continue
		}
		logger.Infof("Full Scan: Found %d topics on page %s", len(topicsOnPage), pageURL)
		tracker.AddTopicsFound(len(topicsOnPage))

		for _, t := range topicsOnPage {
			if _, exists := scannedTopics[t.ID]; !exists {
				scannedTopics[t.ID] = t
			}
		}
		// Politeness delay between processing each page in the loop
		if pageCounter < len(pageURLs) { // No delay after the very last page in this loop
			logger.Debugf("Full Scan: Applying %dms delay after processing page %d/%d...", requestDelayMs, pageCounter, len(pageURLs))
			time.Sleep(time.Duration(requestDelayMs) * time.Millisecond)
		}
		tracker.LogETC() // Log ETC after processing each page
	}
	return scannedTopics, pageURLs, nil
}

// Placeholder for the existing two-pass logic found in the original main.
// This will be refactored and integrated properly as per Story 1.6 tasks.
// func runLegacyTwoPassLogic(startURL string) { // This function will be removed
// ... (contents of runLegacyTwoPassLogic - will be integrated into main or new functions)
// }

func main() {
	cfg := loadConfig() // Load config first

	// --- Setup Logging (Task 4 from Story 1.6) ---
	// Ensure the output directory exists
	if err := os.MkdirAll(cfg.outputDir, os.ModePerm); err != nil {
		log.Fatalf("Failed to create output directory %s: %v", cfg.outputDir, err)
	}

	logFilePath := filepath.Join(cfg.outputDir, "indexer_run_forum48.log")
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		// Fallback to stderr only if file opening fails
		logger.Init(cfg.logLevel, nil) // Initialize with configured level to stderr
		logger.Errorf("Failed to open log file %s: %v. Logging to stderr only.", logFilePath, err)
	} else {
		// 정상적인 경우: 파일과 stderr 모두에 로깅
		multiWriter := io.MultiWriter(os.Stderr, logFile)
		logger.Init(cfg.logLevel, multiWriter) // Initialize with configured level
		// main 함수가 반환될 때 logFile이 닫히도록 예약
		defer func() {
			if err := logFile.Close(); err != nil {
				// stderr로 직접 로깅 시도 (logger 인스턴스가 이 시점에 유효하지 않을 수 있으므로)
				log.Printf("Failed to close log file %s: %v", logFilePath, err)
			}
		}()
	}

	logger.Infof("Starting Project Waypoint Indexer...")
	logger.Infof("Configuration: URL=%s, OutputDir=%s, Delay=%dms, LogLevel=%s, MaxPages=%d",
		cfg.subForumURL, cfg.outputDir, cfg.requestDelay, cfg.logLevel, cfg.maxPages)
	logger.Infof("Logs will also be written to: %s", logFilePath)

	tracker := metrics.NewMetricsTracker()
	// Defer final metrics logging for the very end, even if panics occur (though log.Fatalf will exit)
	// For robust panic handling, a more complex setup might be needed, but this covers normal exit.
	defer tracker.FinalizeAndLogMetrics()

	logger.Infof("Orchestrator: Initializing core components...")
	// TODO: Initialize other modules (metrics, storage) here as needed.

	// --- Story 1.3: Two-Pass Indexing Strategy ---
	logger.Infof("--- Orchestrator: Starting Initial Full Scan Phase (Story 1.1 & 1.2) ---")
	// performFullScan now takes requestDelay from cfg, the metrics tracker, and cfg.maxPages
	fullScanTopics, discoveredPageURLs, err := performFullScan(cfg.subForumURL, cfg.requestDelay, tracker, cfg.maxPages)
	if err != nil {
		logger.Fatalf("Orchestrator: Error during initial full scan: %v", err)
	}
	logger.Infof("--- Orchestrator: Initial Full Scan Phase Completed. Discovered %d unique topics from %d pages ---", len(fullScanTopics), len(discoveredPageURLs))

	// This map will hold the final, de-duplicated list of topics from all passes.
	finalCombinedTopics := make(map[string]topic.TopicInfo)
	for id, t := range fullScanTopics {
		finalCombinedTopics[id] = t
	}

	logger.Infof("--- Orchestrator: Starting First-Page Re-scan Phase (Story 1.3) ---")
	newOrBumpedTopicsCount := 0
	if len(discoveredPageURLs) > 0 {
		firstPageURL := discoveredPageURLs[0] // Assuming ParsePaginationLinks returns URLs in order, with page 1 first.
		logger.Infof("Orchestrator: Re-scanning first page: %s", firstPageURL)

		// Apply delay before fetching the first page for re-scan
		logger.Debugf("Orchestrator: Applying %dms delay before re-scan fetch...", cfg.requestDelay)
		time.Sleep(time.Duration(cfg.requestDelay) * time.Millisecond)

		tracker.IncrementHTTPRequests() // Track request for first-page re-scan
		firstPageHTML, err := navigation.FetchHTML(firstPageURL)
		if err != nil {
			tracker.IncrementFailedRequests()
			logger.Warnf("Orchestrator: Warning - Failed to fetch first page for re-scan (%s): %v. Proceeding with full scan results only.", firstPageURL, err)
		} else {
			tracker.IncrementSuccessfulRequests()
			// Note: This single page re-scan isn't counted in PagesFetched for ETC purposes in the same way as full scan pages.
			// We could add a specific metric for re-scan pages if needed.
			topicsOnFirstPage, err := topic.ExtractTopics(firstPageHTML, firstPageURL) // Story 1.2 integration
			if err != nil {
				logger.Warnf("Orchestrator: Warning - Failed to extract topics from re-scanned first page (%s): %v. Proceeding with full scan results only.", firstPageURL, err)
			} else {
				logger.Infof("Orchestrator: Found %d topics on re-scanned first page %s", len(topicsOnFirstPage), firstPageURL)
				tracker.AddTopicsFound(len(topicsOnFirstPage)) // Add re-scanned topics to raw count
				for _, t := range topicsOnFirstPage {
					if _, exists := finalCombinedTopics[t.ID]; !exists {
						finalCombinedTopics[t.ID] = t
						newOrBumpedTopicsCount++
					} // TODO: Potentially update existing topics if ancillary data changed (Story 1.3 detail)
				}
				logger.Infof("Orchestrator: Identified %d new or bumped topics from the first-page re-scan.", newOrBumpedTopicsCount)
			}
		}
	} else {
		logger.Infof("Orchestrator: No pages discovered in full scan, skipping first-page re-scan.")
	}
	logger.Infof("--- Orchestrator: First-Page Re-scan Phase Completed ---")

	logger.Infof("Orchestrator: Total unique topics discovered across all passes: %d", len(finalCombinedTopics))
	tracker.SetTopicsAddedToStore(len(finalCombinedTopics)) // Set final count of unique topics for metrics

	// Story 1.4: Persistent Topic Index Storage
	if len(finalCombinedTopics) > 0 {
		logger.Infof("Orchestrator: Attempting to save topic index...")
		err = storage.SaveTopicIndex(cfg.outputDir, finalCombinedTopics, cfg.subForumURL)
		if err != nil {
			logger.Fatalf("Orchestrator: Failed to save topic index: %v", err)
		} else {
			logger.Infof("Orchestrator: Topic index saved successfully.")
		}
	} else {
		logger.Infof("Orchestrator: No topics discovered, skipping save operation.")
	}

	// Story 1.5: Performance Metrics & ETC are logged via defer tracker.FinalizeAndLogMetrics()
	// and tracker.LogETC() in performFullScan.

	logger.Infof("Project Waypoint Indexer finished.")
	os.Exit(0) // tracker.FinalizeAndLogMetrics() will be called before exit due to defer
}
