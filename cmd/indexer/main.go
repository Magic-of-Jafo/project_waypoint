package main

import (
	"fmt"
	"log"
	"os"

	"project-waypoint/internal/indexer/navigation"
)

func main() {
	log.Println("Starting sub-forum page indexer...")

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <start_url>", os.Args[0])
	}
	startURL := os.Args[1]

	log.Printf("Fetching initial page: %s\n", startURL)
	htmlContent, err := navigation.FetchHTML(startURL)
	if err != nil {
		log.Fatalf("Failed to fetch HTML from %s: %v", startURL, err)
	}

	log.Println("Parsing pagination links...")
	pageURLs, err := navigation.ParsePaginationLinks(htmlContent, startURL)
	if err != nil {
		log.Fatalf("Failed to parse pagination links from %s: %v", startURL, err)
	}

	log.Printf("Discovered %d page URLs for sub-forum:\n", len(pageURLs))
	for i, pageURL := range pageURLs {
		fmt.Printf("%d: %s\n", i+1, pageURL)
	}

	log.Println("Sub-forum page indexing complete.")
}
