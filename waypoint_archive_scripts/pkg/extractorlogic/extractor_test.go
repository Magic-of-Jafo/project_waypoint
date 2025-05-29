package extractorlogic

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

// Helper function to create a predictable sorted list of ArchivedPageInfo for comparison
func sortPages(pages []ArchivedPageInfo) {
	sort.Slice(pages, func(i, j int) bool {
		if pages[i].SubForumID != pages[j].SubForumID {
			return pages[i].SubForumID < pages[j].SubForumID
		}
		if pages[i].TopicID != pages[j].TopicID {
			return pages[i].TopicID < pages[j].TopicID
		}
		return pages[i].PageNumber < pages[j].PageNumber
	})
}

func TestDiscoverArchivedPages(t *testing.T) {
	// Create a temporary directory structure for testing
	tempRootDir, err := os.MkdirTemp("", "testarchive")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempRootDir)

	// Define expected pages
	expectedPages := []ArchivedPageInfo{}

	// Test case 1: Empty root directory
	pages, err := DiscoverArchivedPages(tempRootDir)
	if err != nil {
		t.Errorf("Test Case 1: DiscoverArchivedPages failed for empty dir: %v", err)
	}
	if len(pages) != 0 {
		t.Errorf("Test Case 1: Expected 0 pages in empty dir, got %d", len(pages))
	}

	// Setup for Test Case 2: Valid structure
	sf1Path := filepath.Join(tempRootDir, "sf1")
	topic1Path := filepath.Join(sf1Path, "topic101")
	topic2Path := filepath.Join(sf1Path, "topic102")
	os.MkdirAll(topic1Path, 0755)
	os.MkdirAll(topic2Path, 0755)

	page1_t101_sf1 := filepath.Join(topic1Path, "page_1.html")
	page2_t101_sf1 := filepath.Join(topic1Path, "page_2.html")
	page1_t102_sf1 := filepath.Join(topic2Path, "page_1.html")
	os.WriteFile(page1_t101_sf1, []byte("dummy content"), 0644)
	os.WriteFile(page2_t101_sf1, []byte("dummy content"), 0644)
	os.WriteFile(page1_t102_sf1, []byte("dummy content"), 0644)

	expectedPages = append(expectedPages,
		ArchivedPageInfo{Path: page1_t101_sf1, SubForumID: "sf1", TopicID: "topic101", PageNumber: 1},
		ArchivedPageInfo{Path: page2_t101_sf1, SubForumID: "sf1", TopicID: "topic101", PageNumber: 2},
		ArchivedPageInfo{Path: page1_t102_sf1, SubForumID: "sf1", TopicID: "topic102", PageNumber: 1},
	)

	sf2Path := filepath.Join(tempRootDir, "sf2")
	topic3Path := filepath.Join(sf2Path, "topic201")
	os.MkdirAll(topic3Path, 0755)
	page1_t201_sf2 := filepath.Join(topic3Path, "page_1.html")
	os.WriteFile(page1_t201_sf2, []byte("dummy content"), 0644)
	expectedPages = append(expectedPages, ArchivedPageInfo{Path: page1_t201_sf2, SubForumID: "sf2", TopicID: "topic201", PageNumber: 1})

	// Add some noise: a file at subforum level, a dir at topic level, a non-html file, a non-page_N.html file
	os.WriteFile(filepath.Join(sf1Path, "random_file.txt"), []byte("noise"), 0644)
	os.Mkdir(filepath.Join(topic1Path, "random_dir"), 0755)
	os.WriteFile(filepath.Join(topic1Path, "other.txt"), []byte("noise"), 0644)
	os.WriteFile(filepath.Join(topic1Path, "page_other.html"), []byte("noise"), 0644)

	sortPages(expectedPages) // Ensure expected order matches function output order

	// Test Case 2: Valid structure with some noise
	pages, err = DiscoverArchivedPages(tempRootDir)
	if err != nil {
		t.Fatalf("Test Case 2: DiscoverArchivedPages failed: %v", err)
	}

	// The DiscoverArchivedPages function already sorts its output.
	if !reflect.DeepEqual(pages, expectedPages) {
		t.Errorf("Test Case 2: Discovered pages do not match expected pages.\nExpected: %+v\nGot:      %+v", expectedPages, pages)
		// Detailed print for easier debugging if order is the issue or content mismatch
		if len(pages) == len(expectedPages) {
			for i := range pages {
				if !reflect.DeepEqual(pages[i], expectedPages[i]) {
					t.Logf("Mismatch at index %d:\nExpected: %+v\nGot:      %+v", i, expectedPages[i], pages[i])
				}
			}
		}
	}

	// Test Case 3: Non-existent root directory (DiscoverArchivedPages should handle this by returning an error)
	_, err = DiscoverArchivedPages(filepath.Join(tempRootDir, "nonexistent_root"))
	if err == nil {
		t.Errorf("Test Case 3: Expected error for non-existent root directory, got nil")
	}

	// Test Case 4: Root directory is a file, not a directory
	fileAsRootDir := filepath.Join(tempRootDir, "file_as_root")
	os.WriteFile(fileAsRootDir, []byte("i am a file"), 0644)
	_, err = DiscoverArchivedPages(fileAsRootDir)
	if err == nil {
		t.Errorf("Test Case 4: Expected error when root is a file, got nil")
	}
	// Note: os.ReadDir returns an error if the path is a file, so this is implicitly tested.

	// Test Case 5: Sub-forum path is a file, not a directory
	sfFileInsteadOfDir := filepath.Join(tempRootDir, "sf_file")
	os.WriteFile(sfFileInsteadOfDir, []byte("file"), 0644)
	// DiscoverArchivedPages should skip this and continue with others if any.
	// In the current implementation, it might return an error for the whole operation if it tries to ReadDir on a file.
	// Let's create a valid one alongside it to test if it processes valid ones.
	validSfPath := filepath.Join(tempRootDir, "sf_valid")
	validTopicPath := filepath.Join(validSfPath, "topic_valid")
	os.MkdirAll(validTopicPath, 0755)
	validPagePath := filepath.Join(validTopicPath, "page_1.html")
	os.WriteFile(validPagePath, []byte("content"), 0644)

	expectedSinglePage := []ArchivedPageInfo{
		{Path: validPagePath, SubForumID: "sf_valid", TopicID: "topic_valid", PageNumber: 1},
	}
	sortPages(expectedSinglePage)

	// Need to clear tempRootDir of previous test files to isolate this test case's dirs for cleaner check
	os.RemoveAll(sf1Path)
	os.RemoveAll(sf2Path)
	// sfFileInsteadOfDir and validSfPath are direct children of tempRootDir now

	pagesAfterFileDirMix, err := DiscoverArchivedPages(tempRootDir)
	if err != nil {
		// Current DiscoverArchivedPages stops on first error in ReadDir sub-loop, so it won't process validSfPath if sf_file is encountered first and causes ReadDir error.
		// Depending on OS, ReadDir on sfFileInsteadOfDir might not error but return empty. The code structure expects IsDir() check.
		// The current code `if !subForumEntry.IsDir() { continue }` handles this gracefully.
		t.Logf("Test Case 5: Note: DiscoverArchivedPages error with mixed file/dir at SF level: %v. This might be acceptable if it skips the file entry.", err)
	}

	// Filter out sfFileInsteadOfDir related entries for comparison if any (should be none from DiscoverArchivedPages)
	actualPagesForCompare := []ArchivedPageInfo{}
	for _, p := range pagesAfterFileDirMix {
		if p.SubForumID != "sf_file" { // sf_file is the one that's a file, not a dir
			actualPagesForCompare = append(actualPagesForCompare, p)
		}
	}
	sortPages(actualPagesForCompare) // Ensure consistent order for comparison

	if !reflect.DeepEqual(actualPagesForCompare, expectedSinglePage) {
		t.Errorf("Test Case 5: Failed. Expected to correctly process valid subforums even with file entries at subforum level.\nExpected: %+v\nGot:      %+v", expectedSinglePage, actualPagesForCompare)
	}
	os.Remove(sfFileInsteadOfDir) // Clean up for next test
	os.RemoveAll(validSfPath)     // Clean up

}

// TestDiscoverArchivedPagesWalkDir can be similarly structured if needed for the alternative func.
// For now, focusing on the primary DiscoverArchivedPages.
