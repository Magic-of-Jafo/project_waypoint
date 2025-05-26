package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	// subForumListCSVPath is the path to the CSV file containing the list of sub-forums.
	subForumListCSVPath = "data/subforum_list.csv"
	// completedSubForumsFilePath is the path to the file that logs IDs of successfully completed sub-forums.
	completedSubForumsFilePath = "data/completed_subforums.txt"
	// coreIndexerExecutableName is the name of the core indexer program.
	// Assumes it's in the same directory or in PATH. For Windows, add .exe if needed.
	coreIndexerExecutableName = "core_indexer.exe" // Or "core_indexer" for Linux/macOS
	// masterOutputBaseDir is the base directory where master_indexer will store outputs from core_indexer.
	masterOutputBaseDir = "master_output/indexed_data"
)

// SubForum holds the information for a sub-forum, read from the CSV
type SubForum struct {
	ID                    string
	Name                  string
	BaseURL               string
	Description           string
	TopicsCount           string
	PostsCount            string
	LastActiveDateTimeStr string
	LastActiveBy          string
	LastPostID            string
}

func loadSubForums(csvFilePath string) ([]SubForum, error) {
	file, err := os.Open(csvFilePath)
	if err != nil {
		return nil, fmt.Errorf("error opening sub-forum CSV file '%s': %w", csvFilePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = 9 // Expect 9 fields based on generate_subforum_list
	reader.Comment = '#'       // Allow comments if any, though not expected from generator

	// Read header row
	_, err = reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("sub-forum CSV file '%s' is empty or has no header", csvFilePath)
		}
		return nil, fmt.Errorf("error reading header from sub-forum CSV file '%s': %w", csvFilePath, err)
	}

	var subForums []SubForum
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break // End of file
			}
			// Consider if we want to skip bad rows or fail
			return nil, fmt.Errorf("error reading record from sub-forum CSV file '%s': %w", csvFilePath, err)
		}

		if len(record) != 9 {
			log.Printf("WARN: Skipping record with incorrect number of fields (%d expected 9): %v", len(record), record)
			continue
		}

		sf := SubForum{
			ID:                    record[0],
			Name:                  record[1],
			BaseURL:               record[2],
			Description:           record[3],
			TopicsCount:           record[4],
			PostsCount:            record[5],
			LastActiveDateTimeStr: record[6],
			LastActiveBy:          record[7],
			LastPostID:            record[8],
		}
		subForums = append(subForums, sf)
	}

	if len(subForums) == 0 {
		return nil, fmt.Errorf("no sub-forum data found in CSV file '%s' (after header)", csvFilePath)
	}

	return subForums, nil
}

// loadCompletedSubForumIDs reads the list of completed sub-forum IDs from the specified file.
// It returns a map for quick lookups. Creates the file if it doesn't exist.
func loadCompletedSubForumIDs(filePath string) (map[string]bool, error) {
	completedIDs := make(map[string]bool)

	file, err := os.OpenFile(filePath, os.O_RDONLY|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening or creating completed IDs file '%s': %w", filePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		id := strings.TrimSpace(scanner.Text())
		if id != "" {
			completedIDs[id] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading completed IDs file '%s': %w", filePath, err)
	}

	return completedIDs, nil
}

// markSubForumAsCompleted appends a sub-forum ID to the completed IDs file.
func markSubForumAsCompleted(filePath string, subForumID string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return fmt.Errorf("error opening or creating completed IDs file '%s' for append: %w", filePath, err)
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, subForumID); err != nil {
		return fmt.Errorf("error writing completed ID '%s' to file '%s': %w", subForumID, filePath, err)
	}
	return nil
}

func main() {
	log.Printf("Master Indexer starting...")

	// Ensure the core_indexer executable exists (basic check)
	// A more robust check might involve `exec.LookPath` or checking specific build output paths.
	if _, err := os.Stat(coreIndexerExecutableName); os.IsNotExist(err) {
		log.Fatalf("FATAL: Core indexer executable '%s' not found. Please build it first (e.g., go build -o %s cmd/indexer/main.go)", coreIndexerExecutableName, coreIndexerExecutableName)
	}

	log.Printf("Loading completed sub-forum IDs from: %s", completedSubForumsFilePath)
	completedIDs, err := loadCompletedSubForumIDs(completedSubForumsFilePath)
	if err != nil {
		log.Fatalf("FATAL: Could not load completed sub-forum IDs: %v", err)
	}
	log.Printf("Loaded %d completed sub-forum IDs.", len(completedIDs))

	log.Printf("Attempting to load sub-forums from: %s", subForumListCSVPath)
	subForums, err := loadSubForums(subForumListCSVPath)
	if err != nil {
		log.Fatalf("FATAL: Could not load sub-forums: %v", err)
	}

	log.Printf("Successfully loaded %d total sub-forums to process.", len(subForums))

	totalSubForums := len(subForums)
	processedCount := 0
	skippedCount := 0

	for i, sf := range subForums {
		log.Printf("--- Processing sub-forum %d/%d: ID=%s, Name=%s ---", i+1, totalSubForums, sf.ID, sf.Name)

		if sf.ID == "" || sf.BaseURL == "" {
			log.Printf("WARN: Sub-forum ID or BaseURL is empty for entry %d. Skipping. CSV Record: %+v", i+1, sf)
			continue
		}

		if completedIDs[sf.ID] {
			log.Printf("Sub-forum %s (%s) already marked as completed. Skipping.", sf.ID, sf.Name)
			skippedCount++
			continue
		}

		// --- Call Core Indexing Script ---
		log.Printf("Invoking core_indexer for sub-forum %s (%s)...", sf.ID, sf.Name)

		// Define the output directory for this specific sub-forum
		subForumOutputDir := filepath.Join(masterOutputBaseDir, "forum_"+sf.ID)
		if err := os.MkdirAll(subForumOutputDir, os.ModePerm); err != nil {
			log.Printf("ERROR: Could not create output directory '%s' for sub-forum %s: %v. Skipping this sub-forum.", subForumOutputDir, sf.ID, err)
			continue
		}
		log.Printf("Core indexer output for forum %s will be in: %s", sf.ID, subForumOutputDir)

		// Construct the command
		// Example: ./core_indexer.exe -url="http://forum.example.com/viewforum.php?f=123" -output="master_output/indexed_data/forum_123" -delay=1000 -loglevel=INFO
		cmdArgs := []string{
			"-url=" + sf.BaseURL,
			"-output=" + subForumOutputDir,
			"-delay=1000",    // Default delay, can be made configurable later
			"-loglevel=INFO", // Default log level, can be made configurable later
			// "-maxpages=1", // Temporarily set for quick testing of the full master_indexer -> core_indexer call - REMOVING FOR FULL RUNS
		}
		cmd := exec.Command("./"+coreIndexerExecutableName, cmdArgs...)
		log.Printf("Executing: %s %s", cmd.Path, strings.Join(cmdArgs, " "))

		// Capture stdout and stderr for logging
		var cmdOut strings.Builder
		var cmdErr strings.Builder
		cmd.Stdout = &cmdOut
		cmd.Stderr = &cmdErr

		processingSuccessful := false
		err := cmd.Run()
		if err != nil {
			log.Printf("ERROR: Core Indexing Script for sub-forum %s (%s) failed. ExitError: %v", sf.ID, sf.Name, err)
			if cmdOut.Len() > 0 {
				log.Printf("Core Indexer STDOUT:\n--- Output Start ---\n%s\n--- Output End ---", cmdOut.String())
			}
			if cmdErr.Len() > 0 {
				log.Printf("Core Indexer STDERR:\n--- Error Output Start ---\n%s\n--- Error Output End ---", cmdErr.String())
			}
			// processingSuccessful remains false
		} else {
			log.Printf("Core Indexing Script for sub-forum %s (%s) completed successfully.", sf.ID, sf.Name)
			if cmdOut.Len() > 0 { // Log stdout even on success, as it might contain useful summary info
				log.Printf("Core Indexer STDOUT:\n--- Output Start ---\n%s\n--- Output End ---", cmdOut.String())
			}
			if cmdErr.Len() > 0 { // Log stderr even on success, in case of non-fatal warnings
				log.Printf("Core Indexer STDERR:\n--- Error Output Start ---\n%s\n--- Error Output End ---", cmdErr.String())
			}
			processingSuccessful = true
		}
		// --- End Call Core Indexing Script ---

		if processingSuccessful {
			err := markSubForumAsCompleted(completedSubForumsFilePath, sf.ID)
			if err != nil {
				log.Printf("ERROR: Failed to mark sub-forum %s (%s) as completed: %v", sf.ID, sf.Name, err)
			} else {
				log.Printf("Successfully marked sub-forum %s (%s) as completed in master log.", sf.ID, sf.Name)
				processedCount++
			}
		} else {
			log.Printf("Processing FAILED for sub-forum %s (%s). It will be retried on the next run.", sf.ID, sf.Name)
		}
		log.Printf("--- Finished processing attempt for sub-forum %s (%s) ---", sf.ID, sf.Name)
	}

	log.Printf("Master Indexer finished.")
	log.Printf("Summary: Total Sub-forums: %d, Processed successfully in this run: %d, Skipped (already complete): %d", totalSubForums, processedCount, skippedCount)
}
