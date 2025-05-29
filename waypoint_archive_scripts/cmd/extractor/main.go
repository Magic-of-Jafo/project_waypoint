package main

import (
	"fmt"
	"log"
	"os"

	"waypoint_archive_scripts/pkg/config"
	"waypoint_archive_scripts/pkg/extractorlogic"
)

func main() {
	// Load configuration
	// Pass os.Args[1:] to allow CLI overrides for config path or other params
	cfg, err := config.LoadConfig(os.Args[1:])
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if cfg.ArchiveRootDir == "" {
		log.Fatalf("ArchiveRootDir is not set in the configuration. Please configure the path to the Waypoint Archive.")
	}

	log.Printf("Starting extractor. Archive root directory: %s", cfg.ArchiveRootDir)

	// Discover archived pages
	archivedPages, err := extractorlogic.DiscoverArchivedPages(cfg.ArchiveRootDir)
	if err != nil {
		log.Fatalf("Error discovering archived pages: %v", err)
	}

	if len(archivedPages) == 0 {
		log.Println("No archived pages found.")
		return
	}

	log.Printf("Discovered %d archived page(s):", len(archivedPages))
	for i, page := range archivedPages {
		// Print only a few to avoid flooding logs if there are many pages
		if i < 10 || i > len(archivedPages)-6 {
			fmt.Printf("  - Path: %s, SubForum: %s, Topic: %s, Page: %d\n",
				page.Path, page.SubForumID, page.TopicID, page.PageNumber)
		} else if i == 10 {
			fmt.Printf("  ... (omitting %d pages) ...\n", len(archivedPages)-15)
		}
	}

	log.Println("Extractor finished discovering pages.")
	// Further steps from Story 3.1 (reading HTML, identifying posts, etc.) will be added here.
}
