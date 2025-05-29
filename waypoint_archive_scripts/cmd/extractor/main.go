package main

import (
	// Will be used if we print individual post details later
	"log"
	"os"

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/extractorlogic"
	"waypoint_archive_scripts/pkg/htmlprocessor"
)

const ( // Log prefixes as per docs/operational-guidelines.md Section 4.4
	logPrefixInfo    = "[INFO]"
	logPrefixWarning = "[WARNING]"
	logPrefixError   = "[ERROR]"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig(os.Args[1:])
	if err != nil {
		// log.Fatalf already prints to os.Stderr and exits; no prefix needed as per guidelines for FATAL
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.ArchiveRootDir == "" {
		log.Fatalf("ArchiveRootDir is not set in the configuration. Please configure the path to the Waypoint Archive.")
	}

	log.Printf("%s Extractor: Starting. Archive root directory: %s", logPrefixInfo, cfg.ArchiveRootDir)

	// Discover archived pages (AC2)
	discoveredPages, err := extractorlogic.DiscoverArchivedPages(cfg.ArchiveRootDir)
	if err != nil {
		log.Fatalf("%s Extractor: Critical error discovering archived pages: %v", logPrefixError, err) // Considered fatal if discovery fails
	}

	if len(discoveredPages) == 0 {
		log.Printf("%s Extractor: No archived pages found to process.", logPrefixInfo)
		return
	}

	log.Printf("%s Extractor: Discovered %d archived page(s). Starting processing loop.", logPrefixInfo, len(discoveredPages))

	processedPageCount := 0
	totalPostBlocksFound := 0
	var pagesWithLoadErrors []string
	var pagesWithPostBlockErrors []string // For errors from GetPostBlocks, if any in future

	for i, pageInfo := range discoveredPages {
		log.Printf("%s Extractor: Processing page %d/%d: %s", logPrefixInfo, i+1, len(discoveredPages), pageInfo.Path)

		// Load and parse the HTML file (AC3)
		htmlPage, err := htmlprocessor.LoadHTMLPage(pageInfo.Path)
		if err != nil {
			log.Printf("%s Extractor: Failed to load/parse HTML page %s: %v. Skipping page.", logPrefixError, pageInfo.Path, err) // AC8
			pagesWithLoadErrors = append(pagesWithLoadErrors, pageInfo.Path)
			continue // Continue with the next file
		}

		// Identify post blocks (AC4, AC5, AC6)
		postBlocks, err := htmlPage.GetPostBlocks()
		if err != nil {
			// This error path in GetPostBlocks is currently not expected as it always returns nil error,
			// but good to have for future changes.
			log.Printf("%s Extractor: Failed to get post blocks from %s: %v. Skipping page.", logPrefixError, pageInfo.Path, err)
			pagesWithPostBlockErrors = append(pagesWithPostBlockErrors, pageInfo.Path)
			continue
		}

		// Log number of post blocks found (AC7)
		log.Printf("%s Extractor: Identified %d post blocks in %s", logPrefixInfo, len(postBlocks), pageInfo.Path)
		totalPostBlocksFound += len(postBlocks)

		// TODO: Future stories will process each postBlock here.
		// For example:
		// for _, block := range postBlocks {
		//   metadata, content, err := ExtractDataFromPostBlock(block)
		//   // ... handle extracted data ...
		//   // log.Printf("%s Extracted post by %s at %s", logPrefixInfo, metadata.User, metadata.Timestamp)
		// }

		processedPageCount++
	}

	log.Printf("%s Extractor: Finished processing loop.", logPrefixInfo)
	log.Printf("%s Extractor Summary: Total pages discovered: %d", logPrefixInfo, len(discoveredPages))
	log.Printf("%s Extractor Summary: Total pages processed successfully for post block identification: %d", logPrefixInfo, processedPageCount)
	log.Printf("%s Extractor Summary: Total post blocks identified: %d", logPrefixInfo, totalPostBlocksFound)

	if len(pagesWithLoadErrors) > 0 {
		log.Printf("%s Extractor Summary: %d page(s) failed to load/parse:", logPrefixWarning, len(pagesWithLoadErrors))
		for _, path := range pagesWithLoadErrors {
			log.Printf("%s Extractor Summary: Failed path: %s", logPrefixWarning, path)
		}
	}
	if len(pagesWithPostBlockErrors) > 0 {
		log.Printf("%s Extractor Summary: %d page(s) had errors getting post blocks (unexpected for current GetPostBlocks impl):", logPrefixWarning, len(pagesWithPostBlockErrors))
		for _, path := range pagesWithPostBlockErrors {
			log.Printf("%s Extractor Summary: Failed path (post block stage): %s", logPrefixWarning, path)
		}
	}
	log.Println("Extractor finished.") // Simple final message
}
