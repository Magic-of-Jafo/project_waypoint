package main

import (
	"fmt"
	"log"
	"os"

	"project-waypoint/internal/indexer/navigation"
	"project-waypoint/internal/indexer/topic"
)

func main() {
	log.Println("Starting sub-forum topic indexer...")

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <start_url>", os.Args[0])
	}
	startURL := os.Args[1]

	log.Printf("Fetching initial page to discover all page URLs: %s\n", startURL)
	initialHTMLContent, err := navigation.FetchHTML(startURL)
	if err != nil {
		log.Fatalf("Failed to fetch HTML from %s: %v", startURL, err)
	}

	log.Println("Parsing pagination links...")
	pageURLs, err := navigation.ParsePaginationLinks(initialHTMLContent, startURL)
	if err != nil {
		log.Fatalf("Failed to parse pagination links from %s: %v", startURL, err)
	}

	log.Printf("Discovered %d page URLs for sub-forum.\n", len(pageURLs))

	allTopics := make(map[string]topic.TopicInfo)
	pageCounter := 0

	for _, pageURL := range pageURLs {
		pageCounter++
		log.Printf("Processing page %d/%d: %s\n", pageCounter, len(pageURLs), pageURL)
		htmlContent, err := navigation.FetchHTML(pageURL)
		if err != nil {
			log.Printf("Failed to fetch HTML from %s: %v. Skipping this page.", pageURL, err)
			continue
		}

		topicsOnPage, err := topic.ExtractTopics(htmlContent, pageURL)
		if err != nil {
			log.Printf("Failed to extract topics from %s: %v. Skipping this page.", pageURL, err)
			continue
		}

		log.Printf("Found %d topics on page %s\n", len(topicsOnPage), pageURL)
		for _, t := range topicsOnPage {
			if _, exists := allTopics[t.ID]; !exists {
				allTopics[t.ID] = t
			}
		}
	}

	log.Printf("Total unique topics discovered across all pages: %d\n", len(allTopics))
	if len(allTopics) > 0 {
		fmt.Println("\n--- All Discovered Topics ---")
		i := 0
		for _, t := range allTopics {
			i++
			fmt.Printf("%d: ID: %s, Title: %s, URL: %s\n", i, t.ID, t.Title, t.URL)
		}
	}

	log.Println("Sub-forum topic indexing complete.")
}
