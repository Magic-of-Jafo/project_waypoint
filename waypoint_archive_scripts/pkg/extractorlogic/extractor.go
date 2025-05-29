package extractorlogic

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// This file will contain the core logic for the data extraction process.

// ArchivedPageInfo holds information about a single archived HTML page.
// This can be expanded later if more info is needed directly from the path.
// For now, just the path is sufficient for Subtask 1.2.
// When implementing Subtask 2.1 (HTML file reading), this struct might evolve
// or a new one created to hold the content or a reader to it.
// For Subtask 2.3 (post block identification), this or another struct will need to
// hold the isolated post blocks.
// For Subtask 3.1 (logging), this will be useful for logging the file path.
type ArchivedPageInfo struct {
	Path       string
	SubForumID string // Extracted from path
	TopicID    string // Extracted from path
	PageNumber int    // Extracted from filename
}

// DiscoverArchivedPages traverses the archive directory structure and finds all HTML page files.
// It expects a structure like: {ARCHIVE_ROOT}/{sub_forum_id_or_name}/{topic_id}/page_{page_number}.html
// It returns a slice of ArchivedPageInfo, sorted for deterministic processing, or an error.
func DiscoverArchivedPages(archiveRootDir string) ([]ArchivedPageInfo, error) {
	var pages []ArchivedPageInfo
	pageFileRegex := regexp.MustCompile(`^page_(\d+)\.html$`)

	// 1. Read sub-forum directories
	subForumDirs, err := os.ReadDir(archiveRootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read archive root directory %s: %w", archiveRootDir, err)
	}

	for _, subForumEntry := range subForumDirs {
		if !subForumEntry.IsDir() {
			continue // Skip non-directory entries at sub-forum level
		}
		subForumID := subForumEntry.Name()
		subForumPath := filepath.Join(archiveRootDir, subForumID)

		// 2. Read topic directories within each sub-forum
		topicDirs, err := os.ReadDir(subForumPath)
		if err != nil {
			// Log or collect error, decide if to continue with other sub-forums
			// For now, let's return the error to halt if a sub-forum dir is unreadable
			return nil, fmt.Errorf("failed to read sub-forum directory %s: %w", subForumPath, err)
		}

		for _, topicEntry := range topicDirs {
			if !topicEntry.IsDir() {
				continue // Skip non-directory entries at topic level
			}
			topicID := topicEntry.Name()
			topicPath := filepath.Join(subForumPath, topicID)

			// 3. Read page files within each topic directory
			pageFiles, err := os.ReadDir(topicPath)
			if err != nil {
				// Log or collect error, decide if to continue with other topics
				return nil, fmt.Errorf("failed to read topic directory %s: %w", topicPath, err)
			}

			for _, pageFileEntry := range pageFiles {
				if pageFileEntry.IsDir() {
					continue // Skip directories, expecting only files like page_N.html
				}

				fileName := pageFileEntry.Name()
				matches := pageFileRegex.FindStringSubmatch(fileName)

				if len(matches) == 2 { // 0 is full string, 1 is the page number capture group
					pageNumber, err := strconv.Atoi(matches[1])
					if err != nil {
						// This should not happen if regex is correct, but good to handle
						// Log this as a warning or error and skip file
						fmt.Printf("[WARNING] Could not parse page number from filename %s in topic %s: %v\n", fileName, topicPath, err)
						continue
					}

					pages = append(pages, ArchivedPageInfo{
						Path:       filepath.Join(topicPath, fileName),
						SubForumID: subForumID,
						TopicID:    topicID,
						PageNumber: pageNumber,
					})
				} else if strings.HasSuffix(strings.ToLower(fileName), ".html") {
					// Log unexpected HTML files that don't match the pattern
					fmt.Printf("[INFO] Found non-standard HTML file: %s in %s\n", fileName, topicPath)
				}
			}
		}
	}

	// Sort pages for deterministic processing. Primarily by SubForumID, then TopicID, then PageNumber.
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].SubForumID != pages[j].SubForumID {
			return pages[i].SubForumID < pages[j].SubForumID
		}
		if pages[i].TopicID != pages[j].TopicID {
			return pages[i].TopicID < pages[j].TopicID
		}
		return pages[i].PageNumber < pages[j].PageNumber
	})

	return pages, nil
}

// Helper function to use filepath.WalkDir for a potentially more robust traversal.
// This version is an alternative and can be developed further if ReadDir proves problematic
// or for deeper hierarchies not expected by this story.
func DiscoverArchivedPagesWalkDir(archiveRootDir string) ([]ArchivedPageInfo, error) {
	var pages []ArchivedPageInfo
	pageFileRegex := regexp.MustCompile(`^page_(\d+)\.html$`)

	err := filepath.WalkDir(archiveRootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Propagate error from WalkDir itself (e.g., permission issues)
			// Potentially log and decide to skip certain paths with errors
			return err
		}

		// We are interested only in files, not directories, at the point of matching page_N.html
		if d.IsDir() {
			// If we are at the archiveRootDir, subForumDir, or topicDir, allow WalkDir to proceed.
			// We could add depth checks if necessary, but for now, the filename check is key.
			// Example: if path == archiveRootDir || (number of path separators indicates it's a subforum or topic dir)
			// For this specific structure {root}/{sf}/{t}/page.html, pages are at depth 3.
			relPath, _ := filepath.Rel(archiveRootDir, path)
			depth := len(strings.Split(filepath.ToSlash(relPath), "/"))
			if d.Name() == "." { // Current directory case for relPath
				depth = 0
			}

			if depth < 3 { // archiveRootDir (0), subForum (1), topic (2)
				return nil // Continue walking
			} else if depth == 3 { // Files within topic dir
				// Let it process files, handled below
			} else {
				return filepath.SkipDir // Too deep, skip this directory
			}
		}

		fileName := d.Name()
		matches := pageFileRegex.FindStringSubmatch(fileName)

		if len(matches) == 2 {
			dir := filepath.Dir(path)
			topicID := filepath.Base(dir)
			subForumID := filepath.Base(filepath.Dir(dir))

			// Basic validation that we are not grabbing files from unexpected places
			// This relies on the depth check being reasonably effective for directories
			expectedParentDir := filepath.Join(archiveRootDir, subForumID, topicID)
			if dir != expectedParentDir {
				fmt.Printf("[WARNING] Mismatched path for %s: expected parent %s, got %s. Skipping.\n", path, expectedParentDir, dir)
				return nil
			}

			pageNumber, parseErr := strconv.Atoi(matches[1])
			if parseErr != nil {
				fmt.Printf("[WARNING] Could not parse page number from filename %s: %v. Skipping.\n", path, parseErr)
				return nil // Skip this file
			}

			pages = append(pages, ArchivedPageInfo{
				Path:       path,
				SubForumID: subForumID,
				TopicID:    topicID,
				PageNumber: pageNumber,
			})
		} else if strings.HasSuffix(strings.ToLower(fileName), ".html") {
			// Log unexpected HTML files not matching the pattern if they are at the expected depth
			relPath, _ := filepath.Rel(archiveRootDir, path)
			if len(strings.Split(filepath.ToSlash(relPath), "/")) == 4 { // file is child of topic dir
				fmt.Printf("[INFO] Found non-standard HTML file: %s\n", path)
			}
		}
		return nil // Continue walking
	})

	if err != nil {
		return nil, fmt.Errorf("error walking the path %s: %w", archiveRootDir, err)
	}

	// Sort pages for deterministic processing
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].SubForumID != pages[j].SubForumID {
			return pages[i].SubForumID < pages[j].SubForumID
		}
		if pages[i].TopicID != pages[j].TopicID {
			return pages[i].TopicID < pages[j].TopicID
		}
		return pages[i].PageNumber < pages[j].PageNumber
	})

	return pages, nil
}
