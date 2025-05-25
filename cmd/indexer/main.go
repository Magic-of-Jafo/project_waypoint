package main

import (
	"fmt"
	"log"
	"os"
	"sort" // New: For ordered output

	// "sort" // Will be needed later for ordered output if desired

	"project-waypoint/internal/indexer/navigation"
	"project-waypoint/internal/indexer/topic"
)

// performFullScan fetches all pages for a given sub-forum URL, extracts topics from each page,
// and returns a map of unique topics found, and the list of page URLs discovered.
func performFullScan(startURL string) (map[string]topic.TopicInfo, []string, error) { // Return pageURLs as well
	log.Printf("Full Scan: Fetching initial page to discover all page URLs: %s\n", startURL)
	initialHTMLContent, err := navigation.FetchHTML(startURL)
	if err != nil {
		return nil, nil, fmt.Errorf("full scan: failed to fetch HTML from %s: %w", startURL, err)
	}

	log.Println("Full Scan: Parsing pagination links...")
	pageURLs, err := navigation.ParsePaginationLinks(initialHTMLContent, startURL)
	if err != nil {
		return nil, nil, fmt.Errorf("full scan: failed to parse pagination links from %s: %w", startURL, err)
	}
	log.Printf("Full Scan: Discovered %d page URLs for sub-forum.\n", len(pageURLs))

	scannedTopics := make(map[string]topic.TopicInfo)
	pageCounter := 0

	for _, pageURL := range pageURLs {
		pageCounter++
		log.Printf("Full Scan: Processing page %d/%d: %s\n", pageCounter, len(pageURLs), pageURL)
		htmlContent, err := navigation.FetchHTML(pageURL)
		if err != nil {
			log.Printf("Full Scan: Failed to fetch HTML from %s: %v. Skipping this page.", pageURL, err)
			continue
		}

		topicsOnPage, err := topic.ExtractTopics(htmlContent, pageURL)
		if err != nil {
			log.Printf("Full Scan: Failed to extract topics from %s: %v. Skipping this page.", pageURL, err)
			continue
		}
		log.Printf("Full Scan: Found %d topics on page %s\n", len(topicsOnPage), pageURL)

		for _, t := range topicsOnPage {
			if _, exists := scannedTopics[t.ID]; !exists {
				scannedTopics[t.ID] = t
			}
		}
	}
	return scannedTopics, pageURLs, nil // Return pageURLs
}

func main() {
	log.Println("Starting sub-forum topic indexer (Two-Pass)...")

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <start_url>", os.Args[0])
	}
	startURL := os.Args[1]

	log.Println("--- Starting Initial Full Scan Phase ---")
	fullScanTopics, discoveredPageURLs, err := performFullScan(startURL)
	if err != nil {
		log.Fatalf("Error during initial full scan: %v", err)
	}
	log.Printf("--- Initial Full Scan Phase Completed: Discovered %d unique topics from %d pages ---", len(fullScanTopics), len(discoveredPageURLs))

	finalCombinedTopics := make(map[string]topic.TopicInfo)
	for id, t := range fullScanTopics {
		finalCombinedTopics[id] = t
	}

	newOrBumpedTopicsCount := 0

	// First-Page Re-scan Phase (AC2, AC3)
	log.Println("--- Starting First-Page Re-scan Phase ---")
	firstPageURL := startURL
	if len(discoveredPageURLs) > 0 {
		firstPageURL = discoveredPageURLs[0]
	}
	log.Printf("Re-scanning first page: %s\n", firstPageURL)
	firstPageHTML, err := navigation.FetchHTML(firstPageURL)
	if err != nil {
		log.Printf("Warning: Failed to fetch first page for re-scan (%s): %v. Proceeding with full scan results only.", firstPageURL, err)
	} else {
		rescanTopicsOnFirstPage, err := topic.ExtractTopics(firstPageHTML, firstPageURL)
		if err != nil {
			log.Printf("Warning: Failed to extract topics from re-scanned first page (%s): %v. Proceeding with full scan results only.", firstPageURL, err)
		} else {
			log.Printf("Found %d topics on re-scanned first page %s\n", len(rescanTopicsOnFirstPage), firstPageURL)

			// Comparison and Final List Compilation (AC4, AC5, AC6) & Logging for New/Bumped (AC9)
			for _, t := range rescanTopicsOnFirstPage {
				if _, exists := finalCombinedTopics[t.ID]; !exists {
					finalCombinedTopics[t.ID] = t
					newOrBumpedTopicsCount++
				}
			}
			log.Printf("Identified %d new or bumped topics from the first-page re-scan.", newOrBumpedTopicsCount)
		}
	}
	log.Println("--- First-Page Re-scan Phase Completed ---")

	log.Printf("Total unique topics discovered across all passes: %d\n", len(finalCombinedTopics))
	if len(finalCombinedTopics) > 0 {
		fmt.Println("\n--- All Discovered Topics (Combined & Sorted by ID) ---")

		// Convert map to slice for sorted output
		var topicsToPrint []topic.TopicInfo
		for _, t := range finalCombinedTopics {
			topicsToPrint = append(topicsToPrint, t)
		}
		// Sort by ID for consistent output order
		sort.Slice(topicsToPrint, func(i, j int) bool {
			return topicsToPrint[i].ID < topicsToPrint[j].ID
		})

		for i, t := range topicsToPrint {
			fmt.Printf("%d: ID: %s, Title: %s, URL: %s\n", i+1, t.ID, t.Title, t.URL)
		}
	}

	log.Println("Sub-forum topic indexing (Two-Pass) complete.")
}
