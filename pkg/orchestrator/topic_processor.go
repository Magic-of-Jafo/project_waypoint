package orchestrator

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"project-waypoint/pkg/data" // Assuming PostMetadata is here
	"project-waypoint/pkg/extractorlogic"
	"project-waypoint/pkg/htmlparser"
	"project-waypoint/pkg/parser" // Added for content parsing

	"github.com/PuerkitoBio/goquery"
)

// TopicInfo might be needed to carry subforum_id or other relevant topic-level details.
// For now, we'll assume subforum_id can be derived or is part of archivePath.
type TopicInfo struct {
	SubforumID string
	TopicID    string
}

// ProcessTopic orchestrates the processing of all pages for a given topic ID.
// It identifies HTML files, processes them in order, and prepares for data extraction.
func ProcessTopic(topicID string, archivePath string, outputPath string /*, topicMetadata data.TopicInfo */) error {
	log.Printf("[INFO] Starting processing for Topic ID: %s", topicID)

	var topicFiles []string
	var derivedSubforumID string // Renamed from subforumID to avoid conflict if TopicInfo is introduced

	// Walk through the archivePath to find the correct topic directory
	// This is a simplified search. A more robust solution might involve a pre-built index or more specific path construction.
	err := filepath.Walk(archivePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasSuffix(path, topicID) { // Found a directory ending with topicID
			// Potential topic directory found. Now look for page_*.html files inside.
			// Extract subforumID from path. Example: /archive/subforum123/topic456 -> subforum123
			parentDir := filepath.Dir(path)
			derivedSubforumID = filepath.Base(parentDir) // Store the derived subforum ID

			pageEntries, err := os.ReadDir(path)
			if err != nil {
				log.Printf("[ERROR] Error reading topic directory %s: %v", path, err)
				return err // Continue walking? Or stop? For now, stop if error reading potential dir.
			}
			for _, entry := range pageEntries {
				if !entry.IsDir() && strings.HasPrefix(entry.Name(), "page_") && strings.HasSuffix(entry.Name(), ".html") {
					topicFiles = append(topicFiles, filepath.Join(path, entry.Name()))
				}
			}
			return filepath.SkipDir // Stop searching further once topic directory is processed
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("error scanning archive for topic %s: %w", topicID, err)
	}

	if len(topicFiles) == 0 {
		return fmt.Errorf("no HTML files found for topic ID %s in %s (derived subforum: %s)", topicID, archivePath, derivedSubforumID)
	}

	// Subtask 1.3: Ensure page files are processed in ascending order of page_number.
	sort.SliceStable(topicFiles, func(i, j int) bool {
		pageNumberI := extractPageNumber(filepath.Base(topicFiles[i]))
		pageNumberJ := extractPageNumber(filepath.Base(topicFiles[j]))
		return pageNumberI < pageNumberJ
	})

	log.Printf("[INFO] Found %d HTML files for topic %s (Subforum: %s). Processing in order.", len(topicFiles), topicID, derivedSubforumID)

	var allPostsForTopic []data.PostMetadata // Store all extracted metadata

	for i, filePath := range topicFiles {
		log.Printf("[INFO] Processing page %d: %s", i+1, filePath)

		// Task 2.1 (part 1): Load HTML page
		page, err := htmlparser.LoadHTMLPage(filePath)
		if err != nil {
			log.Printf("[WARNING] Error loading HTML page %s: %v. Skipping page.", filePath, err)
			continue
		}

		// Task 2.1 (part 2): Identify post blocks
		postBlocks, err := page.GetPostBlocks()
		if err != nil {
			log.Printf("[WARNING] Error getting post blocks from %s: %v. Skipping page.", filePath, err)
			continue
		}

		if len(postBlocks) == 0 {
			log.Printf("[INFO] No post blocks found on page %s.", filePath)
			continue
		}

		log.Printf("[INFO] Found %d post blocks on page %s.", len(postBlocks), filePath)

		for j, postBlock := range postBlocks {
			// Debugging postBlock.Selection itself
			log.Printf("[DEBUG ProcessTopic] PostBlock %d: Selection Length: %d, NodeName: %s",
				j+1, postBlock.Selection.Length(), goquery.NodeName(postBlock.Selection))

			// Get the outer HTML of the <tr> element itself
			postHTML, outerHTMLErr := goquery.OuterHtml(postBlock.Selection)
			if outerHTMLErr != nil {
				log.Printf("[WARNING] Error getting outer HTML for post block %d on page %s: %v", j+1, filePath, outerHTMLErr)
				continue
			}

			// Log the actual string obtained from OuterHtml
			log.Printf("[DEBUG ProcessTopic] OuterHTML string for post %d on page %s:\n%s\n--------------------",
				j+1, filepath.Base(filePath), postHTML)

			// Create a new goquery document from the outer HTML, ensuring it's wrapped for consistent parsing
			// Match the wrapping style of extractor_test.go (no explicit tbody)
			wrappedPostHTML := "<!DOCTYPE html><html><body><table>" + postHTML + "</table></body></html>"
			postDoc, docErr := goquery.NewDocumentFromReader(strings.NewReader(wrappedPostHTML))
			if docErr != nil {
				log.Printf("[WARNING] Error creating wrapped goquery document for post block %d on page %s: %v. Skipping post.",
					j+1, filePath, docErr)
				continue
			}
			// Task 2.2: Extract post metadata
			// Note: The original Story 3.2 signature was ExtractPostMetadata(postHTML HTMLBlock, topicContext Context)
			// The actual function in extractorlogic is ExtractPostMetadata(postHTMLBlock *goquery.Document, filePath string)
			// We are using the latter. `filePath` is used by extractor to get subforum_id, topic_id, page_number.
			metadata, extractErr := extractorlogic.ExtractPostMetadata(postDoc, filePath)
			if extractErr != nil {
				log.Printf("[WARNING] Error extracting metadata for post %d on page %s: %v. Some metadata might be missing.", j+1, filePath, extractErr)
				return fmt.Errorf("fatal error during metadata extraction for post %d on page %s: %w", j+1, filePath, extractErr)
			}

			// Task 3.1: Parse Post Content into Structured Blocks
			// Find the main content cell. The selector for the post content cell is typically td.normal.bgc1.vat.w90
			// ParseContentBlocks expects the selection of the content container itself.
			postContentSelection := postDoc.Find("td.normal.bgc1.vat.w90").First() // Assuming this is the direct container of mixed content nodes.
			if postContentSelection.Length() == 0 {
				log.Printf("[WARNING] Could not find post content cell (td.normal.bgc1.vat.w90) for post %d on page %s. Skipping content parsing.", j+1, filePath)
			} else {
				parsedBlocks, err := parser.ParseContentBlocks(postContentSelection)
				if err != nil {
					log.Printf("[WARNING] Error parsing content blocks for post %d on page %s: %v. Content may be incomplete.", j+1, filePath, err)
				}

				// Task 3.2: Clean NewText Blocks
				for k, block := range parsedBlocks {
					if block.Type == data.ContentBlockTypeNewText {
						cleanedText, cleanErr := parser.CleanNewTextBlock(block.Content) // block.Content is raw HTML here
						if cleanErr != nil {
							log.Printf("[WARNING] Error cleaning new_text block for post %d, block %d: %v. Using raw content.", j+1, k, cleanErr)
						} else {
							parsedBlocks[k].Content = cleanedText
						}
					}
				}
				metadata.ParsedContent = parsedBlocks
			}

			// Task 4.2: Construct PostURL
			if metadata.TopicID != "" && metadata.PostID != "" {
				metadata.PostURL = fmt.Sprintf("viewtopic.php?t=%s&p=%s#p%s", metadata.TopicID, metadata.PostID, metadata.PostID)
			} else {
				log.Printf("[WARNING] Could not construct PostURL for post %d on page %s due to missing TopicID or PostID.", j+1, filePath)
			}

			// Store the extracted metadata (now including parsed content and PostURL)
			allPostsForTopic = append(allPostsForTopic, metadata)
			log.Printf("[INFO] Successfully processed post %d on page %s. PostID: %s, Author: %s", j+1, filePath, metadata.PostID, metadata.AuthorUsername)
		}
	}

	log.Printf("[INFO] Finished processing all pages for topic %s. Total posts extracted: %d", topicID, len(allPostsForTopic))

	// Task 5: Implement JSON File Saving and Naming
	if len(allPostsForTopic) == 0 {
		log.Printf("[INFO] No posts extracted for topic %s. No JSON file will be saved.", topicID)
		return nil
	}

	// Subtask 5.1: Marshal post data to JSON
	jsonData, err := json.MarshalIndent(allPostsForTopic, "", "  ") // Using Indent for readability
	if err != nil {
		return fmt.Errorf("error marshalling topic %s data to JSON: %w", topicID, err)
	}

	// Subtask 5.3: Construct the filename: {subforum_id}_{topic_id}.json
	if derivedSubforumID == "" {
		log.Printf("[WARNING] subforumID not derived for topic %s. Using 'unknown_subforum' in filename.", topicID)
		derivedSubforumID = "unknown_subforum"
	}
	outputFilename := fmt.Sprintf("%s_%s.json", derivedSubforumID, topicID)
	fullOutputPath := filepath.Join(outputPath, outputFilename)

	// Subtask 5.4: Save the JSON string to file
	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, os.ModePerm); err != nil {
		return fmt.Errorf("error creating output directory %s for topic %s: %w", outputPath, topicID, err)
	}

	err = os.WriteFile(fullOutputPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing JSON file %s for topic %s: %w", fullOutputPath, topicID, err)
	}

	// AC14: Log confirmation
	log.Printf("[INFO] Successfully saved structured data for topic %s to %s (%d posts)", topicID, fullOutputPath, len(allPostsForTopic))

	return nil
}

// extractPageNumber extracts the page number from a filename like "page_1.html" or "page_123.html".
// Returns -1 if parsing fails.
func extractPageNumber(filename string) int {
	// Remove "page_" prefix and ".html" suffix
	nameWithoutPrefix := strings.TrimPrefix(filename, "page_")
	nameWithoutSuffix := strings.TrimSuffix(nameWithoutPrefix, ".html")

	pageNumber, err := strconv.Atoi(nameWithoutSuffix)
	if err != nil {
		log.Printf("[WARNING] Could not parse page number from filename '%s': %v", filename, err)
		return -1 // Indicate error or handle as per requirements
	}
	return pageNumber
}

// Mock TopicInfo for now, to be used by ProcessTopic.
// This will eventually be passed in or derived more robustly.
// var currentTopicInfo = TopicInfo{}
